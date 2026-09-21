package einoagent

import (
	"testing"

	"github.com/Mr9esx/Pixoma/internal/studio/domain"
)

func TestApprovalAuthorizerConsumesEachApprovalOnce(t *testing.T) {
	authorizer := newApprovalAuthorizer([]*domain.Approval{{
		Action: "mcp.connector.search.abc", Status: domain.ApprovalApproved,
	}})
	if !authorizer.Consume("mcp.connector.search.abc") {
		t.Fatal("first approved intent was not allowed")
	}
	if authorizer.Consume("mcp.connector.search.abc") {
		t.Fatal("approved intent was reused")
	}
	if authorizer.Consume("mcp.connector.search.different") {
		t.Fatal("different tool intent was allowed")
	}
}
