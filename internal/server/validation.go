package server

import (
	"regexp"
	"strings"
)

var projectNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

func validateSearchRequest(req SearchRequest) error {
	q := strings.TrimSpace(req.Query)
	if q == "" {
		return &ValidationError{Code: "INVALID_QUERY", Message: "query is required and must not be empty"}
	}
	if len(q) > 500 {
		return &ValidationError{Code: "INVALID_QUERY", Message: "query must not exceed 500 characters"}
	}
	if req.Project != "" && !projectNameRegex.MatchString(req.Project) {
		return &ValidationError{Code: "INVALID_PROJECT", Message: "project name contains invalid characters"}
	}
	if req.Limit < 0 || req.Limit > 100 {
		return &ValidationError{Code: "INVALID_LIMIT", Message: "limit must be between 0 and 100"}
	}
	return nil
}

type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func sanitizeProject(name string) string {
	name = strings.TrimSpace(name)
	if !projectNameRegex.MatchString(name) {
		return ""
	}
	return name
}
