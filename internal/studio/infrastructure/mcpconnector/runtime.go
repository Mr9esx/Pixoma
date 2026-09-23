package mcpconnector

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	einojsonschema "github.com/eino-contrib/jsonschema"
	"github.com/google/uuid"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	studioapp "github.com/Mr9esx/Pixoma/internal/studio/application"
	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

const (
	mcpToolTimeout          = 30 * time.Second
	maxMCPToolResultBytes   = 256 << 10
	truncatedToolResultNote = "\n[工具结果过长，已截断]"
)

// ToolAccess connects runtime tool calls to the current Studio run without
// exposing the run repository or any persisted secrets to the MCP package.
type ToolAccess struct {
	PermissionMode  domain.PermissionMode
	IsApproved      func(action string) bool
	RequestApproval func(ctx context.Context, toolCallID, action string) error
	Emit            func(ctx context.Context, eventType string, payload any) error
}

// NewRuntimeTools resolves persisted discovery snapshots into Eino invokable
// tools. The snapshot is an allow-list and contract boundary: remote tool
// changes remain unavailable until the user probes the connector again in AI
// settings.
func NewRuntimeTools(ctx context.Context, connectors []studioapp.ResolvedMCPConnector, access ToolAccess) ([]einotool.BaseTool, error) {
	tools := make([]einotool.BaseTool, 0)
	for _, connector := range connectors {
		if connector.Policy == domain.ConnectorPolicyForbidden || len(connector.Tools) == 0 {
			continue
		}
		for _, discovered := range connector.Tools {
			if strings.TrimSpace(discovered.Name) == "" {
				continue
			}
			params, err := paramsFromMCPTool(discovered.InputSchema)
			if err != nil {
				return nil, fmt.Errorf("studio: decode MCP tool schema %s/%s: %w", connector.ID, discovered.Name, err)
			}
			tools = append(tools, &runtimeTool{
				info: &schema.ToolInfo{
					Name:        runtimeToolName(connector.ID, discovered.Name),
					Desc:        discovered.Description,
					ParamsOneOf: params,
				},
				connector:      connector,
				remoteToolName: discovered.Name,
				access:         access,
			})
		}
	}
	return tools, nil
}

type runtimeTool struct {
	info           *schema.ToolInfo
	connector      studioapp.ResolvedMCPConnector
	remoteToolName string
	access         ToolAccess
}

func (t *runtimeTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *runtimeTool) InvokableRun(ctx context.Context, arguments string, _ ...einotool.Option) (string, error) {
	argumentsValue, err := decodeArguments(arguments)
	if err != nil {
		return "", fmt.Errorf("studio: invalid MCP tool arguments for %s: %w", t.info.Name, err)
	}
	action, err := approvalAction(t.connector.ID, t.remoteToolName, argumentsValue)
	if err != nil {
		return "", err
	}
	if t.requiresApproval() && (t.access.IsApproved == nil || !t.access.IsApproved(action)) {
		if t.access.RequestApproval == nil {
			return "", fmt.Errorf("studio: approval handler is not configured")
		}
		if err := t.access.RequestApproval(ctx, approvalToolCallID(action), action); err != nil {
			return "", err
		}
		return "", compose.Interrupt(ctx, action)
	}
	if err := t.emit(ctx, studioapp.EventToolCallStart, map[string]any{
		"tool_call_id": action, "tool_name": t.info.Name, "connector_id": t.connector.ID,
		"argument_bytes": len(arguments),
	}); err != nil {
		return "", err
	}
	if err := t.emit(ctx, studioapp.EventToolCallArgs, map[string]any{
		"tool_call_id": action, "delta": redactConnectorText(arguments, t.connector.Credential),
	}); err != nil {
		return "", err
	}

	result, err := callRemoteTool(ctx, t.connector, t.remoteToolName, argumentsValue)
	if err != nil {
		safeError := redactConnectorText(err.Error(), t.connector.Credential)
		if emitErr := t.emit(ctx, studioapp.EventToolCallResult, map[string]any{
			"tool_call_id": action, "content": safeError, "is_error": true,
		}); emitErr != nil {
			return "", emitErr
		}
		if emitErr := t.emit(ctx, studioapp.EventToolCallEnd, map[string]any{
			"tool_call_id": action, "tool_name": t.info.Name, "connector_id": t.connector.ID,
			"result_bytes": 0, "is_error": true,
		}); emitErr != nil {
			return "", emitErr
		}
		return "", err
	}
	output, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("studio: encode MCP tool result %s: %w", t.info.Name, err)
	}
	safeOutput := sanitizeToolOutput(output, t.connector.Credential)
	if err := t.emit(ctx, studioapp.EventToolCallResult, map[string]any{
		"tool_call_id": action, "content": safeOutput, "is_error": result.IsError,
	}); err != nil {
		return "", err
	}
	if err := t.emit(ctx, studioapp.EventToolCallEnd, map[string]any{
		"tool_call_id": action, "tool_name": t.info.Name, "connector_id": t.connector.ID,
		"result_bytes": len(safeOutput), "is_error": result.IsError,
	}); err != nil {
		return "", err
	}
	return safeOutput, nil
}

func (t *runtimeTool) requiresApproval() bool {
	return t.connector.Policy == domain.ConnectorPolicyApproval || t.access.PermissionMode == domain.PermissionRequestApproval
}

func (t *runtimeTool) emit(ctx context.Context, eventType string, payload any) error {
	if t.access.Emit == nil {
		return nil
	}
	return t.access.Emit(ctx, eventType, payload)
}

func callRemoteTool(ctx context.Context, connector studioapp.ResolvedMCPConnector, name string, arguments any) (*mcp.CallToolResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, mcpToolTimeout)
	defer cancel()
	session, err := connect(callCtx, connector)
	if err != nil {
		return nil, redactConnectorError(err, connector.Credential)
	}
	defer session.Close()
	result, err := session.CallTool(callCtx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return nil, redactConnectorError(err, connector.Credential)
	}
	return result, nil
}

func connect(ctx context.Context, connector studioapp.ResolvedMCPConnector) (*mcp.ClientSession, error) {
	client := cloneClientWithBearer(&http.Client{Timeout: mcpToolTimeout}, connector.Credential)
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "pixoma-studio", Version: "1.0"}, nil)
	return mcpClient.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: connector.URL, HTTPClient: client, MaxRetries: -1, DisableStandaloneSSE: true,
	}, nil)
}

func paramsFromMCPTool(encoded []byte) (*schema.ParamsOneOf, error) {
	var parameters einojsonschema.Schema
	if err := json.Unmarshal(encoded, &parameters); err != nil {
		return nil, err
	}
	return schema.NewParamsOneOfByJSONSchema(&parameters), nil
}

func decodeArguments(arguments string) (any, error) {
	if strings.TrimSpace(arguments) == "" {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal([]byte(arguments), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func approvalAction(connectorID, toolName string, arguments any) (string, error) {
	canonicalArguments, err := json.Marshal(arguments)
	if err != nil {
		return "", fmt.Errorf("studio: encode MCP tool approval arguments: %w", err)
	}
	digest := sha256.Sum256(canonicalArguments)
	return "mcp." + connectorID + "." + toolName + "." + fmt.Sprintf("%x", digest), nil
}

func approvalToolCallID(action string) string {
	return action + "." + uuid.NewString()
}

func runtimeToolName(connectorID, remoteName string) string {
	return "mcp_" + sanitizeToolName(connectorID) + "_" + sanitizeToolName(remoteName)
}

func sanitizeToolName(value string) string {
	var out strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			out.WriteRune(r)
			continue
		}
		out.WriteByte('_')
	}
	return strings.Trim(out.String(), "_")
}

func redactConnectorError(err error, credential string) error {
	if err == nil {
		return nil
	}
	message := redactConnectorText(err.Error(), credential)
	return fmt.Errorf("studio: MCP connector request failed: %s", message)
}

func sanitizeToolOutput(output []byte, credential string) string {
	safe := redactConnectorText(string(output), credential)
	if len(safe) <= maxMCPToolResultBytes {
		return safe
	}
	limit := maxMCPToolResultBytes - len(truncatedToolResultNote)
	for limit > 0 && !utf8.ValidString(safe[:limit]) {
		limit--
	}
	return safe[:limit] + truncatedToolResultNote
}

func redactConnectorText(value, credential string) string {
	if strings.TrimSpace(credential) == "" {
		return value
	}
	value = strings.ReplaceAll(value, credential, "[REDACTED]")
	escaped, err := json.Marshal(credential)
	if err == nil && len(escaped) >= 2 {
		value = strings.ReplaceAll(value, string(escaped[1:len(escaped)-1]), "[REDACTED]")
	}
	for _, encoded := range []string{
		base64.StdEncoding.EncodeToString([]byte(credential)),
		base64.RawStdEncoding.EncodeToString([]byte(credential)),
		base64.URLEncoding.EncodeToString([]byte(credential)),
		base64.RawURLEncoding.EncodeToString([]byte(credential)),
		url.QueryEscape(credential),
	} {
		if encoded != "" {
			value = strings.ReplaceAll(value, encoded, "[REDACTED]")
		}
	}
	return value
}
