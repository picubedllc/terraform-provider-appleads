// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Common Apple Ads response header names.
const (
	HeaderRequestID = "X-Request-Id"
	HeaderAPContext = "X-AP-Context"
)

// APIError is a typed Apple Ads API error suitable for Terraform diagnostics.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	Body       string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "apple ads api error: status=%d", e.StatusCode)
	if e.Code != "" {
		fmt.Fprintf(&b, " code=%s", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, " message=%s", e.Message)
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, " request_id=%s", e.RequestID)
	}
	return b.String()
}

type appleErrorBody struct {
	Error struct {
		Errors []struct {
			MessageCode string `json:"messageCode"`
			Message     string `json:"message"`
			Field       string `json:"field"`
		} `json:"errors"`
	} `json:"error"`
	// Some responses use a flatter shape.
	MessageCode string `json:"messageCode"`
	Message     string `json:"message"`
}

func parseAPIError(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		RequestID:  firstHeader(resp, HeaderRequestID, "X-Request-ID", "Request-Id"),
		Body:       string(body),
	}

	var parsed appleErrorBody
	if len(body) > 0 && json.Unmarshal(body, &parsed) == nil {
		if len(parsed.Error.Errors) > 0 {
			apiErr.Code = parsed.Error.Errors[0].MessageCode
			apiErr.Message = parsed.Error.Errors[0].Message
			if parsed.Error.Errors[0].Field != "" && apiErr.Message != "" {
				apiErr.Message = parsed.Error.Errors[0].Field + ": " + apiErr.Message
			}
		} else {
			apiErr.Code = parsed.MessageCode
			apiErr.Message = parsed.Message
		}
	}

	if apiErr.Message == "" && len(body) > 0 {
		apiErr.Message = strings.TrimSpace(string(body))
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}

func firstHeader(resp *http.Response, names ...string) string {
	for _, name := range names {
		if v := resp.Header.Get(name); v != "" {
			return v
		}
	}
	return ""
}
