package jira

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPlainTextToADF_TaskListLocalIDs_Unique(t *testing.T) {
	got := plainTextToADF("- [ ] a\n\n- [x] b")
	content := got["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 task lists, got %d", len(content))
	}
	seen := map[string]bool{}
	for _, block := range content {
		list := block.(map[string]any)
		if list["type"] != "taskList" {
			t.Fatalf("expected taskList, got %v", list["type"])
		}
		attrs := list["attrs"].(map[string]any)
		localID, _ := attrs["localId"].(string)
		if localID == "" {
			t.Fatalf("taskList missing localId: %v", list)
		}
		if seen[localID] {
			t.Fatalf("duplicate taskList localId %q", localID)
		}
		seen[localID] = true
		items := list["content"].([]any)
		itemAttrs := items[0].(map[string]any)["attrs"].(map[string]any)
		itemID, _ := itemAttrs["localId"].(string)
		if itemID == "" {
			t.Fatalf("taskItem missing localId: %v", items[0])
		}
		if seen[itemID] {
			t.Fatalf("duplicate taskItem localId %q", itemID)
		}
		seen[itemID] = true
	}
}

func TestPlainTextToADF(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single line one paragraph",
			in:   "hello",
			want: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]}`,
		},
		{
			name: "two lines two paragraphs",
			in:   "line one\nline two",
			want: `{"type":"doc","version":1,"content":[
				{"type":"paragraph","content":[{"type":"text","text":"line one"}]},
				{"type":"paragraph","content":[{"type":"text","text":"line two"}]}
			]}`,
		},
		{
			name: "headings levels",
			in:   "# One\n## Two\n### Three",
			want: `{"type":"doc","version":1,"content":[
				{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"One"}]},
				{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Two"}]},
				{"type":"heading","attrs":{"level":3},"content":[{"type":"text","text":"Three"}]}
			]}`,
		},
		{
			name: "bullet list grouped",
			in:   "- a\n- b",
			want: `{"type":"doc","version":1,"content":[
				{"type":"bulletList","content":[
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]}]},
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"b"}]}]}
				]}
			]}`,
		},
		{
			name: "task list with states and localIds",
			in:   "- [ ] a\n- [x] b",
			want: `{"type":"doc","version":1,"content":[
				{"type":"taskList","attrs":{"localId":"task-list-1"},"content":[
					{"type":"taskItem","attrs":{"localId":"task-1","state":"TODO"},"content":[{"type":"text","text":"a"}]},
					{"type":"taskItem","attrs":{"localId":"task-2","state":"DONE"},"content":[{"type":"text","text":"b"}]}
				]}
			]}`,
		},
		{
			name: "inline bold",
			in:   "foo **bar** baz",
			want: `{"type":"doc","version":1,"content":[
				{"type":"paragraph","content":[
					{"type":"text","text":"foo "},
					{"type":"text","text":"bar","marks":[{"type":"strong"}]},
					{"type":"text","text":" baz"}
				]}
			]}`,
		},
		{
			name: "unbalanced bold falls back to plain text",
			in:   "foo **bar",
			want: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"foo **bar"}]}]}`,
		},
		{
			name: "mixed blocks",
			in:   "# Title\n- a\n- b\nsome paragraph\n- [x] done",
			want: `{"type":"doc","version":1,"content":[
				{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Title"}]},
				{"type":"bulletList","content":[
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]}]},
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"b"}]}]}
				]},
				{"type":"paragraph","content":[{"type":"text","text":"some paragraph"}]},
				{"type":"taskList","attrs":{"localId":"task-list-1"},"content":[
					{"type":"taskItem","attrs":{"localId":"task-1","state":"DONE"},"content":[{"type":"text","text":"done"}]}
				]}
			]}`,
		},
		{
			name: "blank line terminates list group",
			in:   "- a\n\n- b",
			want: `{"type":"doc","version":1,"content":[
				{"type":"bulletList","content":[
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"a"}]}]}
				]},
				{"type":"bulletList","content":[
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"b"}]}]}
				]}
			]}`,
		},
		{
			name: "bullet and task lists not grouped together",
			in:   "- [x] a\n- b",
			want: `{"type":"doc","version":1,"content":[
				{"type":"taskList","attrs":{"localId":"task-list-1"},"content":[
					{"type":"taskItem","attrs":{"localId":"task-1","state":"DONE"},"content":[{"type":"text","text":"a"}]}
				]},
				{"type":"bulletList","content":[
					{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"b"}]}]}
				]}
			]}`,
		},
		{
			name: "crlf line endings trimmed",
			in:   "a\r\nb",
			want: `{"type":"doc","version":1,"content":[
				{"type":"paragraph","content":[{"type":"text","text":"a"}]},
				{"type":"paragraph","content":[{"type":"text","text":"b"}]}
			]}`,
		},
		{
			name: "empty string empty content",
			in:   "",
			want: `{"type":"doc","version":1,"content":[]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := plainTextToADF(tc.in)

			gotJSON, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("marshal got: %v", err)
			}

			var gotNorm, wantNorm any
			if err := json.Unmarshal(gotJSON, &gotNorm); err != nil {
				t.Fatalf("unmarshal got: %v", err)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantNorm); err != nil {
				t.Fatalf("unmarshal want: %v", err)
			}
			if !reflect.DeepEqual(gotNorm, wantNorm) {
				t.Errorf("plainTextToADF(%q) mismatch:\n got: %s\n want: %s", tc.in, gotJSON, tc.want)
			}
		})
	}
}
