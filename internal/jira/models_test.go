package jira

import (
	"encoding/json"
	"testing"
)

func TestCreateIssueRequest_Marshal(t *testing.T) {
	req := CreateIssueRequest{
		ProjectKey:  "PROJ",
		IssueType:   "Task",
		Summary:     "A sample issue",
		Description: "Detailed description",
		Fields: map[string]any{
			"customfield_10001": "value",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	fields := raw["fields"].(map[string]any)
	if fields["project"].(map[string]any)["key"] != "PROJ" {
		t.Errorf("expected project key PROJ, got %v", fields["project"])
	}
	if fields["issuetype"].(map[string]any)["name"] != "Task" {
		t.Errorf("expected issue type Task, got %v", fields["issuetype"])
	}
	if fields["summary"] != "A sample issue" {
		t.Errorf("expected summary, got %v", fields["summary"])
	}
	if fields["description"] != "Detailed description" {
		t.Errorf("expected description, got %v", fields["description"])
	}
	if fields["customfield_10001"] != "value" {
		t.Errorf("expected custom field, got %v", fields["customfield_10001"])
	}
}

func TestUpdateIssueRequest_Marshal(t *testing.T) {
	req := UpdateIssueRequest{
		Summary:     "Updated summary",
		Description: "Updated description",
		Fields: map[string]any{
			"labels": []string{"bug"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	fields := raw["fields"].(map[string]any)
	if fields["summary"] != "Updated summary" {
		t.Errorf("expected summary, got %v", fields["summary"])
	}
	if fields["description"] != "Updated description" {
		t.Errorf("expected description, got %v", fields["description"])
	}
	labels := fields["labels"].([]any)
	if len(labels) != 1 || labels[0] != "bug" {
		t.Errorf("expected labels, got %v", labels)
	}
}

func TestSearchResult_Unmarshal(t *testing.T) {
	payload := `{
		"startAt": 0,
		"maxResults": 50,
		"total": 1,
		"issues": [
			{"id": "10001", "key": "PROJ-1", "self": "https://jira/rest/api/3/issue/10001"}
		]
	}`

	var res SearchResult
	if err := json.Unmarshal([]byte(payload), &res); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if res.Total != 1 {
		t.Errorf("expected total 1, got %d", res.Total)
	}
	if len(res.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(res.Issues))
	}
	if res.Issues[0].Key != "PROJ-1" {
		t.Errorf("expected key PROJ-1, got %s", res.Issues[0].Key)
	}
}

func TestIssue_Unmarshal(t *testing.T) {
	payload := `{
		"id": "10001",
		"key": "PROJ-1",
		"self": "https://jira/rest/api/3/issue/10001",
		"fields": {"summary": "Test summary"}
	}`

	var issue Issue
	if err := json.Unmarshal([]byte(payload), &issue); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if issue.Key != "PROJ-1" {
		t.Errorf("expected key PROJ-1, got %s", issue.Key)
	}

	var fields map[string]any
	if err := json.Unmarshal(issue.Fields, &fields); err != nil {
		t.Fatalf("unmarshal fields failed: %v", err)
	}
	if fields["summary"] != "Test summary" {
		t.Errorf("expected summary, got %v", fields["summary"])
	}
}

func TestTransition_Unmarshal(t *testing.T) {
	payload := `{
		"id": "21",
		"name": "In Progress",
		"to": {"id": "3", "name": "In Progress"}
	}`

	var tr Transition
	if err := json.Unmarshal([]byte(payload), &tr); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if tr.ID != "21" {
		t.Errorf("expected id 21, got %s", tr.ID)
	}
	if tr.Name != "In Progress" {
		t.Errorf("expected name In Progress, got %s", tr.Name)
	}
}

func TestProject_Unmarshal(t *testing.T) {
	payload := `{
		"id": "10000",
		"key": "PROJ",
		"name": "Project",
		"self": "https://jira/rest/api/3/project/10000"
	}`

	var p Project
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if p.Key != "PROJ" {
		t.Errorf("expected key PROJ, got %s", p.Key)
	}
	if p.Name != "Project" {
		t.Errorf("expected name Project, got %s", p.Name)
	}
}

func TestTransitionIssueRequest_Marshal(t *testing.T) {
	req := TransitionIssueRequest{TransitionID: "21"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if raw["transition"].(map[string]any)["id"] != "21" {
		t.Errorf("expected transition id 21, got %v", raw["transition"])
	}
}

func TestAssignIssueRequest_Marshal(t *testing.T) {
	req := AssignIssueRequest{AccountID: "abc123"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if raw["accountId"] != "abc123" {
		t.Errorf("expected accountId abc123, got %v", raw["accountId"])
	}
}

func TestAddCommentRequest_Marshal(t *testing.T) {
	req := AddCommentRequest{Body: "A comment"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if raw["body"] != "A comment" {
		t.Errorf("expected body, got %v", raw["body"])
	}
}

func TestAddWorklogRequest_Marshal(t *testing.T) {
	req := AddWorklogRequest{
		TimeSpent: "1h",
		Comment:   "Worked on issue",
		Started:   "2026-07-16T12:00:00.000+0000",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if raw["timeSpent"] != "1h" {
		t.Errorf("expected timeSpent, got %v", raw["timeSpent"])
	}
	if raw["comment"] != "Worked on issue" {
		t.Errorf("expected comment, got %v", raw["comment"])
	}
	if raw["started"] != "2026-07-16T12:00:00.000+0000" {
		t.Errorf("expected started, got %v", raw["started"])
	}
}

func TestBoard_Unmarshal(t *testing.T) {
	payload := `{
		"id": 1,
		"self": "https://jira/rest/agile/1.0/board/1",
		"name": "Board 1",
		"type": "scrum",
		"location": {"projectId": 10000, "projectKey": "PROJ", "name": "Project"}
	}`

	var b Board
	if err := json.Unmarshal([]byte(payload), &b); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if b.ID != 1 {
		t.Errorf("expected id 1, got %d", b.ID)
	}
	if b.Name != "Board 1" {
		t.Errorf("expected name Board 1, got %s", b.Name)
	}
	if b.Type != "scrum" {
		t.Errorf("expected type scrum, got %s", b.Type)
	}
	if b.Location == nil || b.Location.ProjectKey != "PROJ" {
		t.Errorf("expected location project key PROJ, got %+v", b.Location)
	}
}

func TestBoardListResponse_Unmarshal(t *testing.T) {
	payload := `{
		"maxResults": 50,
		"startAt": 0,
		"isLast": true,
		"values": [
			{"id": 1, "name": "Board 1", "type": "scrum"},
			{"id": 2, "name": "Board 2", "type": "kanban"}
		]
	}`

	var res BoardListResponse
	if err := json.Unmarshal([]byte(payload), &res); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(res.Values) != 2 {
		t.Fatalf("expected 2 boards, got %d", len(res.Values))
	}
	if res.Values[0].Name != "Board 1" {
		t.Errorf("expected Board 1, got %s", res.Values[0].Name)
	}
	if res.Values[1].Type != "kanban" {
		t.Errorf("expected kanban type, got %s", res.Values[1].Type)
	}
}

func TestSprint_Unmarshal(t *testing.T) {
	payload := `{
		"id": 123,
		"self": "https://jira/rest/agile/1.0/sprint/123",
		"state": "active",
		"name": "Sprint 1",
		"startDate": "2026-07-01T10:00:00.000Z",
		"endDate": "2026-07-15T10:00:00.000Z",
		"originBoardId": 42
	}`

	var sp Sprint
	if err := json.Unmarshal([]byte(payload), &sp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if sp.ID != 123 {
		t.Errorf("expected id 123, got %d", sp.ID)
	}
	if sp.Name != "Sprint 1" {
		t.Errorf("expected name Sprint 1, got %s", sp.Name)
	}
	if sp.State != "active" {
		t.Errorf("expected state active, got %s", sp.State)
	}
	if sp.OriginBoardID != 42 {
		t.Errorf("expected origin board id 42, got %d", sp.OriginBoardID)
	}
}

func TestSprintListResponse_Unmarshal(t *testing.T) {
	payload := `{
		"maxResults": 50,
		"startAt": 0,
		"isLast": true,
		"values": [
			{"id": 1, "name": "Sprint 1", "state": "active"},
			{"id": 2, "name": "Sprint 2", "state": "future"}
		]
	}`

	var res SprintListResponse
	if err := json.Unmarshal([]byte(payload), &res); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(res.Values) != 2 {
		t.Fatalf("expected 2 sprints, got %d", len(res.Values))
	}
	if res.Values[0].Name != "Sprint 1" {
		t.Errorf("expected Sprint 1, got %s", res.Values[0].Name)
	}
	if res.Values[1].State != "future" {
		t.Errorf("expected future state, got %s", res.Values[1].State)
	}
}
