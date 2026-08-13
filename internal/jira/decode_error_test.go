package jira

import (
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestJiraErrorResponse_Messages(t *testing.T) {
	cases := []struct {
		name string
		in   jiraErrorResponse
		want []string
	}{
		{
			name: "only errorMessages",
			in:   jiraErrorResponse{ErrorMessages: []string{"first", "second"}},
			want: []string{"first", "second"},
		},
		{
			name: "only errors map sorted by key",
			in: jiraErrorResponse{Errors: map[string]string{
				"summary": "Field 'summary' is required",
				"labels":  "labels is not valid",
			}},
			want: []string{"labels: labels is not valid", "summary: Field 'summary' is required"},
		},
		{
			name: "errorMessages then errors map",
			in: jiraErrorResponse{
				ErrorMessages: []string{"boom"},
				Errors: map[string]string{
					"summary": "Field 'summary' is required",
				},
			},
			want: []string{"boom", "summary: Field 'summary' is required"},
		},
		{
			name: "empty",
			in:   jiraErrorResponse{},
			want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.messages()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("messages() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestDecodeError_SurfacesFieldErrors(t *testing.T) {
	body := `{"errors":{"summary":"Field 'summary' is required"}}`
	resp := &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
	}

	err := decodeError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "jira error 400: validation failed: summary: Field 'summary' is required"
	if err.Error() != want {
		t.Errorf("decodeError() = %q, want %q", err.Error(), want)
	}
}

func TestDecodeError_FallsBackOnBadBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader("not json")),
		Header:     http.Header{},
	}

	err := decodeError(resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "jira error 404: resource not found" {
		t.Errorf("decodeError() = %q, want fallback message", err.Error())
	}
}
