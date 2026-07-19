package jira

import (
	"fmt"
	"strings"
)

// JiraError carries an HTTP status code and a message from the Jira API.
type JiraError struct {
	StatusCode int
	Message    string
}

// Error returns a safe, non-credential-leaking error string.
func (e *JiraError) Error() string {
	return fmt.Sprintf("jira error %d: %s", e.StatusCode, e.Message)
}

// MapHTTPError converts a Jira HTTP status code into a domain error.
// It never includes credentials or request bodies.
func MapHTTPError(statusCode int, messages []string, retryAfter string) error {
	msg := strings.Join(messages, ", ")
	switch statusCode {
	case 400:
		if msg == "" {
			msg = "bad request"
		}
		return &JiraError{StatusCode: statusCode, Message: "validation failed: " + msg}
	case 401:
		return &JiraError{StatusCode: statusCode, Message: "authentication failed"}
	case 403:
		return &JiraError{StatusCode: statusCode, Message: "permission denied"}
	case 404:
		return &JiraError{StatusCode: statusCode, Message: "resource not found"}
	case 429:
		if retryAfter != "" {
			return &JiraError{StatusCode: statusCode, Message: "rate limited; retry after " + retryAfter}
		}
		return &JiraError{StatusCode: statusCode, Message: "rate limited"}
	case 500:
		return &JiraError{StatusCode: statusCode, Message: "jira server error"}
	default:
		return &JiraError{StatusCode: statusCode, Message: fmt.Sprintf("jira request failed (status %d)", statusCode)}
	}
}

// IsRetryableStatus reports whether an idempotent read should be retried once.
func IsRetryableStatus(status int) bool {
	return status == 500 || status == 502 || status == 503
}
