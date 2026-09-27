package einoagent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestInstallSkillToolInfo(t *testing.T) {
	info, err := installSkillToolInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "install_skill" || info.ParamsOneOf == nil || !strings.Contains(info.Desc, "用户明确要求") {
		t.Fatalf("Tool info = %#v", info)
	}
	tool := &installSkillTool{info: info}
	registered, err := tool.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if registered.Name != "install_skill" || registered.ParamsOneOf == nil {
		t.Fatalf("registered Tool = %#v", registered)
	}
	parametersSchema, err := registered.ParamsOneOf.ToJSONSchema()
	if err != nil {
		t.Fatal(err)
	}
	parameters, err := json.Marshal(parametersSchema)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(parameters), "github_url") {
		t.Fatalf("Tool parameters = %s", parameters)
	}
}
