package mcp

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestCreateIssueTool_DescriptionSchema_OneOf(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	tools := srv.mcp.ListTools()
	tool, ok := tools["jira_create_issue"]
	if !ok {
		t.Fatal("missing jira_create_issue tool")
	}

	// Marshal the whole tool to get the wire-shape inputSchema (the raw bytes
	// for tools built via NewToolWithRawSchema are not reflected in the
	// typed InputSchema field).
	schemaBytes := getInputSchema(t, tool.Tool)
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("unmarshal InputSchema: %v", err)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong type: %v", schema["properties"])
	}
	desc, ok := props["description"].(map[string]any)
	if !ok {
		t.Fatalf("description property missing: %v", props)
	}
	oneOf, ok := desc["oneOf"].([]any)
	if !ok {
		t.Fatalf("description.oneOf missing: %v", desc)
	}
	if len(oneOf) != 2 {
		t.Fatalf("expected 2 oneOf branches, got %d: %v", len(oneOf), oneOf)
	}
	b0 := oneOf[0].(map[string]any)
	if b0["type"] != "string" {
		t.Errorf("branch 0 type: got %v want string", b0["type"])
	}
	b1 := oneOf[1].(map[string]any)
	if b1["type"] != "object" {
		t.Errorf("branch 1 type: got %v want object", b1["type"])
	}
	required, ok := b1["required"].([]any)
	if !ok {
		t.Fatalf("branch 1 required missing: %v", b1)
	}
	want := map[string]bool{"type": false, "version": false, "content": false}
	for _, r := range required {
		s, _ := r.(string)
		if _, ok := want[s]; ok {
			want[s] = true
		}
	}
	for k, v := range want {
		if !v {
			t.Errorf("branch 1 required missing %q: %v", k, required)
		}
	}
}

func TestUpdateIssueTool_DescriptionSchema_OneOf(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	tools := srv.mcp.ListTools()
	tool, ok := tools["jira_update_issue"]
	if !ok {
		t.Fatal("missing jira_update_issue tool")
	}
	schemaBytes := getInputSchema(t, tool.Tool)
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("unmarshal InputSchema: %v", err)
	}
	props := schema["properties"].(map[string]any)
	desc := props["description"].(map[string]any)
	oneOf, ok := desc["oneOf"].([]any)
	if !ok {
		t.Fatalf("description.oneOf missing: %v", desc)
	}
	if len(oneOf) != 2 {
		t.Fatalf("expected 2 oneOf branches, got %d", len(oneOf))
	}
	if oneOf[0].(map[string]any)["type"] != "string" {
		t.Errorf("branch 0 type: got %v", oneOf[0])
	}
	if oneOf[1].(map[string]any)["type"] != "object" {
		t.Errorf("branch 1 type: got %v", oneOf[1])
	}
}

func TestCreateIssueTool_ToolDescriptionMentionsADF(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	tools := srv.mcp.ListTools()
	tool := tools["jira_create_issue"]
	if !contains(tool.Tool.Description, "ADF") {
		t.Errorf("tool description should mention ADF: %q", tool.Tool.Description)
	}
}

func TestAddCommentTool_BodySchema_OneOf(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	tools := srv.mcp.ListTools()
	tool, ok := tools["jira_add_comment"]
	if !ok {
		t.Fatal("missing jira_add_comment tool")
	}
	schemaBytes := getInputSchema(t, tool.Tool)
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("unmarshal InputSchema: %v", err)
	}
	props := schema["properties"].(map[string]any)
	body := props["body"].(map[string]any)
	oneOf, ok := body["oneOf"].([]any)
	if !ok {
		t.Fatalf("body.oneOf missing: %v", body)
	}
	if len(oneOf) != 2 {
		t.Fatalf("expected 2 oneOf branches, got %d", len(oneOf))
	}
	if oneOf[0].(map[string]any)["type"] != "string" {
		t.Errorf("branch 0 type: got %v", oneOf[0])
	}
	if oneOf[1].(map[string]any)["type"] != "object" {
		t.Errorf("branch 1 type: got %v", oneOf[1])
	}
}

func TestAddCommentTool_ToolDescriptionMentionsADF(t *testing.T) {
	srv := NewServer(&fakeJiraClient{}, nil)
	tools := srv.mcp.ListTools()
	tool := tools["jira_add_comment"]
	if !contains(tool.Tool.Description, "ADF") {
		t.Errorf("tool description should mention ADF: %q", tool.Tool.Description)
	}
}

// getInputSchema returns the wire-shape inputSchema bytes for a tool, working
// for both typed-builder tools and raw-schema tools.
func getInputSchema(t *testing.T, tool mcp.Tool) []byte {
	t.Helper()
	raw, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("marshal tool: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal tool: %v", err)
	}
	schema, ok := m["inputSchema"].(map[string]any)
	if !ok {
		t.Fatalf("inputSchema missing: %v", m)
	}
	bs, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal inputSchema: %v", err)
	}
	return bs
}
