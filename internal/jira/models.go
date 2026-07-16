package jira

import "encoding/json"

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
	JQL        string   `json:"jql"`
	MaxResults int      `json:"maxResults,omitempty"`
	Fields     []string `json:"fields,omitempty"`
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

// CreateIssueRequest is the payload used to create a new issue.
type CreateIssueRequest struct {
	ProjectKey  string         `json:"-"`
	IssueType   string         `json:"-"`
	Summary     string         `json:"-"`
	Description string         `json:"-"`
	Fields      map[string]any `json:"-"`
}

// MarshalJSON encodes the request into the shape Jira expects.
func (r CreateIssueRequest) MarshalJSON() ([]byte, error) {
	fields := map[string]any{
		"project":   map[string]any{"key": r.ProjectKey},
		"issuetype": map[string]any{"name": r.IssueType},
		"summary":   r.Summary,
	}
	if r.Description != "" {
		fields["description"] = r.Description
	}
	for k, v := range r.Fields {
		fields[k] = v
	}
	return json.Marshal(map[string]any{"fields": fields})
}

// UpdateIssueRequest is the payload used to update an issue.
type UpdateIssueRequest struct {
	Summary     string         `json:"-"`
	Description string         `json:"-"`
	Fields      map[string]any `json:"-"`
}

// MarshalJSON encodes the update payload into the shape Jira expects.
func (r UpdateIssueRequest) MarshalJSON() ([]byte, error) {
	fields := map[string]any{}
	if r.Summary != "" {
		fields["summary"] = r.Summary
	}
	if r.Description != "" {
		fields["description"] = r.Description
	}
	for k, v := range r.Fields {
		fields[k] = v
	}
	return json.Marshal(map[string]any{"fields": fields})
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

// jiraErrorResponse is the shape of Jira error payloads.
type jiraErrorResponse struct {
	ErrorMessages   []string `json:"errorMessages"`
	WarningMessages []string `json:"warningMessages"`
}
