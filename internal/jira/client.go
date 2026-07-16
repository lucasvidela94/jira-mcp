package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lucasvidela/jira-mcp/internal/config"
)

// Client is a thin, stateless Jira Cloud REST API client.
type Client struct {
	baseURL string
	client  *http.Client
	auth    string
}

// Option customizes a client request.
type Option func(*url.Values)

// WithStartAt sets the pagination offset.
func WithStartAt(startAt int) Option {
	return func(v *url.Values) {
		v.Set("startAt", strconv.Itoa(startAt))
	}
}

// WithMaxResults sets the pagination limit.
func WithMaxResults(max int) Option {
	return func(v *url.Values) {
		v.Set("maxResults", strconv.Itoa(max))
	}
}

// New creates a Jira client from configuration.
// If httpClient is nil, a default client with a 30s timeout is used.
func New(cfg config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := strings.TrimRight(cfg.URL, "/")
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.APIToken))
	return &Client{
		baseURL: baseURL,
		client:  httpClient,
		auth:    "Basic " + auth,
	}
}

// do performs an HTTP request against the Jira API.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("invalid jira URL: %w", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira request failed: %w", err)
	}
	return resp, nil
}

// decodeError parses a Jira error response into a domain error.
func decodeError(resp *http.Response) error {
	defer resp.Body.Close()
	var jerr jiraErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&jerr); err != nil {
		return MapHTTPError(resp.StatusCode, nil, resp.Header.Get("Retry-After"))
	}
	return MapHTTPError(resp.StatusCode, jerr.ErrorMessages, resp.Header.Get("Retry-After"))
}

// doWithRetry executes idempotent read requests with one retry on retryable statuses.
func (c *Client) doWithRetry(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	resp, err := c.do(ctx, method, path, query, body)
	if err != nil {
		return nil, err
	}
	if IsRetryableStatus(resp.StatusCode) {
		_ = resp.Body.Close()
		resp, err = c.do(ctx, method, path, query, body)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// handleResponse decodes a successful response or returns a mapped error.
func handleResponse(resp *http.Response, out any) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return decodeError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// Search performs a JQL search.
func (c *Client) Search(ctx context.Context, jql string, opts ...Option) (*SearchResult, error) {
	query := url.Values{}
	query.Set("jql", jql)
	for _, opt := range opts {
		opt(&query)
	}
	resp, err := c.doWithRetry(ctx, http.MethodGet, "/rest/api/3/search", query, nil)
	if err != nil {
		return nil, err
	}
	var result SearchResult
	if err := handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetIssue fetches a single issue by key.
func (c *Client) GetIssue(ctx context.Context, key string) (*Issue, error) {
	resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key)), nil, nil)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := handleResponse(resp, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// ListProjects returns accessible projects.
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	resp, err := c.doWithRetry(ctx, http.MethodGet, "/rest/api/3/project", nil, nil)
	if err != nil {
		return nil, err
	}
	var projects []Project
	if err := handleResponse(resp, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// GetTransitions returns available transitions for an issue.
func (c *Client) GetTransitions(ctx context.Context, key string) ([]Transition, error) {
	resp, err := c.doWithRetry(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(key)), nil, nil)
	if err != nil {
		return nil, err
	}
	var result TransitionsResponse
	if err := handleResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Transitions, nil
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*Issue, error) {
	resp, err := c.do(ctx, http.MethodPost, "/rest/api/3/issue", nil, req)
	if err != nil {
		return nil, err
	}
	var issue Issue
	if err := handleResponse(resp, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// UpdateIssue updates an existing issue.
func (c *Client) UpdateIssue(ctx context.Context, key string, req *UpdateIssueRequest) error {
	resp, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key)), nil, req)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

// TransitionIssue transitions an issue.
func (c *Client) TransitionIssue(ctx context.Context, key string, req *TransitionIssueRequest) error {
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/rest/api/3/issue/%s/transitions", url.PathEscape(key)), nil, req)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

// AssignIssue assigns an issue to a user.
func (c *Client) AssignIssue(ctx context.Context, key string, req *AssignIssueRequest) error {
	resp, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/rest/api/3/issue/%s/assignee", url.PathEscape(key)), nil, req)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

// DeleteIssue deletes an issue.
func (c *Client) DeleteIssue(ctx context.Context, key string) error {
	resp, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(key)), nil, nil)
	if err != nil {
		return err
	}
	return handleResponse(resp, nil)
}

// AddComment adds a comment to an issue.
func (c *Client) AddComment(ctx context.Context, key string, req *AddCommentRequest) (*CommentResponse, error) {
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/rest/api/3/issue/%s/comment", url.PathEscape(key)), nil, req)
	if err != nil {
		return nil, err
	}
	var comment CommentResponse
	if err := handleResponse(resp, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// AddWorklog adds a worklog to an issue.
func (c *Client) AddWorklog(ctx context.Context, key string, req *AddWorklogRequest) (*WorklogResponse, error) {
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/rest/api/3/issue/%s/worklog", url.PathEscape(key)), nil, req)
	if err != nil {
		return nil, err
	}
	var worklog WorklogResponse
	if err := handleResponse(resp, &worklog); err != nil {
		return nil, err
	}
	return &worklog, nil
}
