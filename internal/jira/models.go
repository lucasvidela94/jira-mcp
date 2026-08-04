package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// Issue represents a Jira issue returned by the REST API.
type Issue struct {
	ID     string          `json:"id"`
	Key    string          `json:"key"`
	Self   string          `json:"self"`
	Fields json.RawMessage `json:"fields"`
}

// SearchResult is the response payload from the Jira search endpoint.
// The new /rest/api/3/search/jql endpoint uses nextPageToken for pagination.
type SearchResult struct {
	StartAt       int     `json:"startAt,omitempty"`
	MaxResults    int     `json:"maxResults,omitempty"`
	Total         int     `json:"total,omitempty"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	IsLast        bool    `json:"isLast,omitempty"`
	Issues        []Issue `json:"issues"`
}

// SearchRequest is the POST body for the /rest/api/3/search/jql endpoint.
type SearchRequest struct {
	JQL           string   `json:"jql"`
	MaxResults    int      `json:"maxResults,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// Project is a minimal Jira project representation.
type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Self string `json:"self"`
}

// Transition represents a single available transition for an issue.
type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"to"`
}

// TransitionsResponse wraps the list of available transitions.
type TransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// plainTextToADF wraps a plain string in Atlassian Document Format so that Jira
// accepts it as a description. Reuses the same the same ADF structure for every
// call — one doc node, one paragraph, one text node.
func plainTextToADF(text string) map[string]any {
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{
						"type": "text",
						"text": text,
					},
				},
			},
		},
	}
}

// resolveDescription turns the polymorphic Description into a value that
// belongs inside fields.description, or returns ok=false when nothing should be
// written. The mutual-exclusion with Fields["description"] is enforced here
// so every marshaler that uses Description shares the same rule.
//
// Description is json.RawMessage: when its first byte is a JSON string quote
// ('"'), it carries a plain-text payload that must be wrapped via
// plainTextToADF; otherwise it is a pre-built ADF document that must be
// embedded verbatim. The verbatim branch is returned as json.RawMessage so the
// caller can splice it into a json.Marshal output without re-encoding.
func resolveDescription(description json.RawMessage, fields map[string]any) (json.RawMessage, bool, error) {
	if len(description) == 0 {
		if _, has := fields["description"]; has {
			return nil, false, fmt.Errorf("description provided in fields overrides top-level description: do not set both")
		}
		return nil, false, nil
	}
	if _, has := fields["description"]; has {
		return nil, false, fmt.Errorf("description provided in both top-level argument and fields: choose one")
	}
	if description[0] == '"' {
		var s string
		if err := json.Unmarshal(description, &s); err != nil {
			return nil, false, fmt.Errorf("description: %w", err)
		}
		adf, err := json.Marshal(plainTextToADF(s))
		if err != nil {
			return nil, false, fmt.Errorf("description: %w", err)
		}
		return adf, true, nil
	}
	// Already an ADF object — embed verbatim. Return the raw bytes so the
	// outer marshaler does not re-encode and corrupt byte-equality.
	return description, true, nil
}

// toRaw marshals any value to json.RawMessage, used to populate a
// map[string]json.RawMessage without losing the raw-bytes path for description.
func toRaw(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}

// marshalFieldsWrapper encodes {"fields": <fields>} as JSON, preserving the
// byte-equal property of any pre-marshaled values inside fields. It does not
// re-marshal the inner map; instead it sorts the keys (so output is
// deterministic) and writes raw bytes via an encoder on a buffer.
func marshalFieldsWrapper(fields map[string]json.RawMessage) ([]byte, error) {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString(`{"fields":{`)
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(fields[k])
	}
	buf.WriteString(`}}`)
	return buf.Bytes(), nil
}

// CreateIssueRequest is the payload used to create a new issue.
//
// Description is polymorphic: it can hold a JSON-encoded plain string
// (`"hello"`) which is wrapped via plainTextToADF, or a pre-built ADF
// document object (e.g. `{"type":"doc",...}`) which is embedded verbatim.
// The marshaler branches on the first byte of Description to decide between
// the two paths. Callers must not also set Fields["description"]; the
// marshaler returns an error if both are present.
type CreateIssueRequest struct {
	ProjectKey  string          `json:"-"`
	IssueType   string          `json:"-"`
	Summary     string          `json:"-"`
	Description json.RawMessage `json:"-"`
	Fields      map[string]any  `json:"-"`
}

// MarshalJSON encodes the request into the shape Jira expects.
//
// The fields map is built as map[string]json.RawMessage so the verbatim ADF
// path stays byte-equal to the caller's input. The Fields merge loop
// re-encodes any user-supplied map values, but that was the pre-existing
// behaviour and is not part of the polymorphism contract.
func (r CreateIssueRequest) MarshalJSON() ([]byte, error) {
	fields := map[string]json.RawMessage{}
	for _, kv := range []struct {
		key string
		val any
	}{
		{"project", map[string]any{"key": r.ProjectKey}},
		{"issuetype", map[string]any{"name": r.IssueType}},
		{"summary", r.Summary},
	} {
		raw, err := toRaw(kv.val)
		if err != nil {
			return nil, err
		}
		fields[kv.key] = raw
	}
	desc, ok, err := resolveDescription(r.Description, r.Fields)
	if err != nil {
		return nil, err
	}
	if ok {
		fields["description"] = desc
	}
	for k, v := range r.Fields {
		raw, err := toRaw(v)
		if err != nil {
			return nil, err
		}
		fields[k] = raw
	}
	return marshalFieldsWrapper(fields)
}

// UpdateIssueRequest is the payload used to update an issue.
// See CreateIssueRequest for the polymorphic Description contract.
type UpdateIssueRequest struct {
	Summary     string          `json:"-"`
	IssueType   string          `json:"-"`
	Description json.RawMessage `json:"-"`
	Fields      map[string]any  `json:"-"`
}

// MarshalJSON encodes the update payload into the shape Jira expects.
// See CreateIssueRequest.MarshalJSON for the map[string]json.RawMessage rationale.
func (r UpdateIssueRequest) MarshalJSON() ([]byte, error) {
	fields := map[string]json.RawMessage{}
	if r.Summary != "" {
		summaryRaw, err := toRaw(r.Summary)
		if err != nil {
			return nil, err
		}
		fields["summary"] = summaryRaw
	}
	if r.IssueType != "" {
		raw, err := toRaw(map[string]any{"name": r.IssueType})
		if err != nil {
			return nil, err
		}
		fields["issuetype"] = raw
	}
	desc, ok, err := resolveDescription(r.Description, r.Fields)
	if err != nil {
		return nil, err
	}
	if ok {
		fields["description"] = desc
	}
	for k, v := range r.Fields {
		raw, err := toRaw(v)
		if err != nil {
			return nil, err
		}
		fields[k] = raw
	}
	return marshalFieldsWrapper(fields)
}

// TransitionIssueRequest is the payload used to transition an issue.
type TransitionIssueRequest struct {
	TransitionID string `json:"-"`
}

// MarshalJSON encodes the transition request payload.
func (r TransitionIssueRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"transition": map[string]any{"id": r.TransitionID},
	})
}

// AssignIssueRequest is the payload used to assign an issue.
type AssignIssueRequest struct {
	AccountID string `json:"accountId"`
}

// AddCommentRequest is the payload used to add a comment.
type AddCommentRequest struct {
	Body string `json:"body"`
}

// CommentResponse is the Jira response after adding a comment.
type CommentResponse struct {
	ID   string `json:"id"`
	Self string `json:"self"`
	Body string `json:"body"`
}

// AddWorklogRequest is the payload used to add a worklog.
type AddWorklogRequest struct {
	TimeSpent string `json:"timeSpent,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Started   string `json:"started,omitempty"`
}

// WorklogResponse is the Jira response after adding a worklog.
type WorklogResponse struct {
	ID        string `json:"id"`
	Self      string `json:"self"`
	TimeSpent string `json:"timeSpent"`
}

// Board represents a Jira Agile board.
type Board struct {
	ID       int    `json:"id"`
	Self     string `json:"self"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location *struct {
		ProjectID   int    `json:"projectId,omitempty"`
		DisplayName string `json:"displayName,omitempty"`
		Name        string `json:"name,omitempty"`
		AvatarURI   string `json:"avatarURI,omitempty"`
		ProjectKey  string `json:"projectKey,omitempty"`
	} `json:"location,omitempty"`
}

// BoardList is a slice of Board values used when a handler returns multiple boards.
type BoardList []Board

// BoardListResponse is the paginated response from the Jira Agile board list endpoint.
type BoardListResponse struct {
	MaxResults int       `json:"maxResults"`
	StartAt    int       `json:"startAt"`
	IsLast     bool      `json:"isLast"`
	Values     BoardList `json:"values"`
}

// Sprint represents a Jira Agile sprint.
type Sprint struct {
	ID            int    `json:"id"`
	Self          string `json:"self"`
	State         string `json:"state"`
	Name          string `json:"name"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
	Goal          string `json:"goal,omitempty"`
}

// SprintList is a slice of Sprint values used when a handler returns multiple sprints.
type SprintList []Sprint

// SprintListResponse is the paginated response from the Jira Agile sprint list endpoints.
type SprintListResponse struct {
	MaxResults int        `json:"maxResults"`
	StartAt    int        `json:"startAt"`
	IsLast     bool       `json:"isLast"`
	Values     SprintList `json:"values"`
}

// Changelog is the Jira issue changelog response.
type Changelog struct {
	StartAt    int              `json:"startAt,omitempty"`
	MaxResults int              `json:"maxResults,omitempty"`
	Total      int              `json:"total,omitempty"`
	Histories  []ChangelogEntry `json:"histories"`
}

// ChangelogEntry is a single entry in the issue changelog.
type ChangelogEntry struct {
	ID      string          `json:"id"`
	Author  *User           `json:"author,omitempty"`
	Created string          `json:"created"`
	Items   []ChangelogItem `json:"items"`
}

// ChangelogItem is a single field change in a changelog entry.
type ChangelogItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype"`
	From       string `json:"from,omitempty"`
	FromString string `json:"fromString,omitempty"`
	To         string `json:"to,omitempty"`
	ToString   string `json:"toString,omitempty"`
}

// User is a minimal Jira user representation.
type User struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress,omitempty"`
}

// Version is a Jira project version.
type Version struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Archived    bool   `json:"archived"`
	Released    bool   `json:"released"`
	ReleaseDate string `json:"releaseDate,omitempty"`
	StartDate   string `json:"startDate,omitempty"`
	Overdue     bool   `json:"overdue,omitempty"`
	ProjectID   int    `json:"projectId,omitempty"`
	Self        string `json:"self"`
}

// DevelopmentInformation holds linked dev data from the development field.
type DevelopmentInformation struct {
	Detail *DevelopmentDetail `json:"detail,omitempty"`
}

// DevelopmentDetail holds the dev-info per repository.
type DevelopmentDetail struct {
	Repositories []Repository `json:"repositories,omitempty"`
}

// Repository is a Git repository with linked PRs, branches, and commits.
type Repository struct {
	Name         string        `json:"name,omitempty"`
	URL          string        `json:"url,omitempty"`
	PullRequests []PullRequest `json:"pullRequests,omitempty"`
	Branches     []Branch      `json:"branches,omitempty"`
	Commits      []Commit      `json:"commits,omitempty"`
}

// PullRequest is a pull request linked to an issue.
type PullRequest struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
	Status string `json:"status,omitempty"`
	Open   bool   `json:"open,omitempty"`
	Closed bool   `json:"closed,omitempty"`
	Merged bool   `json:"merged,omitempty"`
}

// Branch is a branch linked to an issue.
type Branch struct {
	Name                 string `json:"name"`
	URL                  string `json:"url,omitempty"`
	CreatePullRequestURL string `json:"createPullRequestUrl,omitempty"`
}

// Commit is a commit linked to an issue.
type Commit struct {
	ID              string `json:"id"`
	Message         string `json:"message,omitempty"`
	URL             string `json:"url,omitempty"`
	AuthorTimestamp string `json:"authorTimestamp,omitempty"`
}

// StatusCategory groups issue statuses by category.
type StatusCategory struct {
	Self      string `json:"self"`
	ID        int    `json:"id"`
	Key       string `json:"key"`
	ColorName string `json:"colorName"`
	Name      string `json:"name"`
}

// Status is a Jira issue status.
type Status struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

// IssueTypeStatuses groups statuses by issue type.
type IssueTypeStatuses struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Self     string   `json:"self,omitempty"`
	Statuses []Status `json:"statuses"`
}

// ProjectStatus is an element of the project statuses response.
type ProjectStatus struct {
	Self     string              `json:"self"`
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Subjects []IssueTypeStatuses `json:"subjects,omitempty"`
	Statuses []Status            `json:"statuses,omitempty"`
}

// IssueLinkRequest is the payload for creating an issue link.
type IssueLinkRequest struct {
	LinkTypeName string `json:"-"`
	InwardKey    string `json:"-"`
	OutwardKey   string `json:"-"`
}

func (r IssueLinkRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":         map[string]any{"name": r.LinkTypeName},
		"inwardIssue":  map[string]any{"key": r.InwardKey},
		"outwardIssue": map[string]any{"key": r.OutwardKey},
	})
}

// IssueLinkType describes a link type between issues.
type IssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
	Self    string `json:"self,omitempty"`
}

// IssueLink is a link between two issues.
type IssueLink struct {
	ID           string        `json:"id,omitempty"`
	Self         string        `json:"self,omitempty"`
	Type         IssueLinkType `json:"type"`
	InwardIssue  *LinkedIssue  `json:"inwardIssue,omitempty"`
	OutwardIssue *LinkedIssue  `json:"outwardIssue,omitempty"`
}

// LinkedIssue is a referenced issue inside a link.
type LinkedIssue struct {
	ID     string          `json:"id"`
	Key    string          `json:"key"`
	Self   string          `json:"self"`
	Fields json.RawMessage `json:"fields,omitempty"`
}

// CreateChildIssueRequest is the payload for creating a child issue.
// See CreateIssueRequest for the polymorphic Description contract.
type CreateChildIssueRequest struct {
	ParentKey   string
	ProjectKey  string
	IssueType   string
	Summary     string
	Description json.RawMessage
	Fields      map[string]any
}

// MarshalJSON encodes the child-issue request into the shape Jira expects.
// See CreateIssueRequest.MarshalJSON for the map[string]json.RawMessage rationale.
func (r CreateChildIssueRequest) MarshalJSON() ([]byte, error) {
	fields := map[string]json.RawMessage{}
	for _, kv := range []struct {
		key string
		val any
	}{
		{"project", map[string]any{"key": r.ProjectKey}},
		{"issuetype", map[string]any{"name": r.IssueType}},
		{"summary", r.Summary},
		{"parent", map[string]any{"key": r.ParentKey}},
	} {
		raw, err := toRaw(kv.val)
		if err != nil {
			return nil, err
		}
		fields[kv.key] = raw
	}
	desc, ok, err := resolveDescription(r.Description, r.Fields)
	if err != nil {
		return nil, err
	}
	if ok {
		fields["description"] = desc
	}
	for k, v := range r.Fields {
		raw, err := toRaw(v)
		if err != nil {
			return nil, err
		}
		fields[k] = raw
	}
	return marshalFieldsWrapper(fields)
}

// parseIssueLinks extracts linked issues from raw issue fields.
func parseIssueLinks(fields json.RawMessage) ([]LinkedIssue, error) {
	if fields == nil {
		return []LinkedIssue{}, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(fields, &raw); err != nil {
		return nil, err
	}
	linksRaw, ok := raw["issuelinks"]
	if !ok {
		return []LinkedIssue{}, nil
	}
	linksJSON, err := json.Marshal(linksRaw)
	if err != nil {
		return nil, err
	}
	var links []IssueLink
	if err := json.Unmarshal(linksJSON, &links); err != nil {
		return nil, err
	}
	var result []LinkedIssue
	for _, l := range links {
		if l.InwardIssue != nil {
			result = append(result, *l.InwardIssue)
		}
		if l.OutwardIssue != nil {
			result = append(result, *l.OutwardIssue)
		}
	}
	return result, nil
}

// parseDevelopmentInfo extracts development information from raw issue fields.
func parseDevelopmentInfo(fields json.RawMessage) (*DevelopmentInformation, error) {
	if fields == nil {
		return &DevelopmentInformation{}, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(fields, &raw); err != nil {
		return nil, err
	}
	devRaw, ok := raw["development"]
	if !ok {
		return &DevelopmentInformation{}, nil
	}
	devJSON, err := json.Marshal(devRaw)
	if err != nil {
		return nil, err
	}
	var info DevelopmentInformation
	if err := json.Unmarshal(devJSON, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// jiraErrorResponse is the shape of Jira error payloads.
type jiraErrorResponse struct {
	ErrorMessages   []string `json:"errorMessages"`
	WarningMessages []string `json:"warningMessages"`
}
