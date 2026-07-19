package jira

import (
	"testing"
)

func TestJiraError_Error(t *testing.T) {
	err := &JiraError{StatusCode: 404, Message: "issue not found"}
	if err.Error() != "jira error 404: issue not found" {
		t.Errorf("unexpected error string: %q", err.Error())
	}
}

func TestMapHTTPError(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		messages   []string
		retryAfter string
		want       string
	}{
		{
			name:       "400 validation",
			statusCode: 400,
			messages:   []string{"Invalid JQL"},
			want:       "jira error 400: validation failed: Invalid JQL",
		},
		{
			name:       "401 authentication",
			statusCode: 401,
			want:       "jira error 401: authentication failed",
		},
		{
			name:       "403 permission",
			statusCode: 403,
			want:       "jira error 403: permission denied",
		},
		{
			name:       "404 not found",
			statusCode: 404,
			want:       "jira error 404: resource not found",
		},
		{
			name:       "429 rate limited",
			statusCode: 429,
			retryAfter: "60",
			want:       "jira error 429: rate limited; retry after 60",
		},
		{
			name:       "500 server error",
			statusCode: 500,
			want:       "jira error 500: jira server error",
		},
		{
			name:       "502 unknown",
			statusCode: 502,
			want:       "jira error 502: jira request failed (status 502)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapHTTPError(tc.statusCode, tc.messages, tc.retryAfter)
			if got.Error() != tc.want {
				t.Errorf("MapHTTPError() = %q, want %q", got.Error(), tc.want)
			}
		})
	}
}

func TestIsRetryableStatus(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   bool
	}{
		{"500", 500, true},
		{"502", 502, true},
		{"503", 503, true},
		{"504", 504, false},
		{"400", 400, false},
		{"404", 404, false},
		{"429", 429, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsRetryableStatus(tc.status); got != tc.want {
				t.Errorf("IsRetryableStatus(%d) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}
