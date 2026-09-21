package mcpconnector

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestApprovalActionBindsCanonicalToolArguments(t *testing.T) {
	first, err := approvalAction("connector", "search", map[string]any{"query": "rain", "limit": float64(3)})
	if err != nil {
		t.Fatal(err)
	}
	equivalent, err := approvalAction("connector", "search", map[string]any{"limit": float64(3), "query": "rain"})
	if err != nil {
		t.Fatal(err)
	}
	different, err := approvalAction("connector", "search", map[string]any{"query": "sun", "limit": float64(3)})
	if err != nil {
		t.Fatal(err)
	}
	if first != equivalent {
		t.Fatalf("equivalent arguments produced different actions: %q != %q", first, equivalent)
	}
	if first == different {
		t.Fatalf("different arguments produced the same action: %q", first)
	}
}

func TestSanitizeToolOutputRedactsAndBoundsResult(t *testing.T) {
	output := sanitizeToolOutput([]byte(strings.Repeat("x", maxMCPToolResultBytes*2)+"connector-secret"), "connector-secret")
	if strings.Contains(output, "connector-secret") {
		t.Fatal("credential leaked from tool output")
	}
	if len(output) > maxMCPToolResultBytes {
		t.Fatalf("tool output length = %d, exceeds %d", len(output), maxMCPToolResultBytes)
	}
	if !strings.HasSuffix(output, truncatedToolResultNote) {
		t.Fatal("large tool output was not marked as truncated")
	}
}

func TestRedactConnectorTextRedactsCommonCredentialEncodings(t *testing.T) {
	credential := "connector-secret"
	value := credential + " " + base64.StdEncoding.EncodeToString([]byte(credential))
	redacted := redactConnectorText(value, credential)
	if strings.Contains(redacted, credential) || strings.Contains(redacted, base64.StdEncoding.EncodeToString([]byte(credential))) {
		t.Fatalf("credential encoding leaked: %q", redacted)
	}
}
