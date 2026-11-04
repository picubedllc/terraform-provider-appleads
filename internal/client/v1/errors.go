// SPDX-License-Identifier: MIT

package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

// APIError is an alias so callers can use v1.APIError while sharing the type.
type APIError = client.APIError

type platformErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Info    struct {
				Field string `json:"field"`
			} `json:"info"`
		} `json:"details"`
	} `json:"error"`
	// Legacy / transitional flat fields.
	Code    string `json:"code"`
	Message string `json:"message"`
}

func parseAPIError(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		RequestID:  firstHeader(resp, client.HeaderRequestID, "X-Request-ID", "Request-Id"),
		Body:       string(body),
	}

	var parsed platformErrorBody
	if len(body) > 0 && json.Unmarshal(body, &parsed) == nil {
		apiErr.Code = firstNonEmpty(parsed.Error.Code, parsed.Code)
		apiErr.Message = firstNonEmpty(parsed.Error.Message, parsed.Message)
		if len(parsed.Error.Details) > 0 {
			d := parsed.Error.Details[0]
			if d.Code != "" {
				apiErr.Code = d.Code
			}
			msg := d.Message
			if d.Info.Field != "" && msg != "" {
				msg = d.Info.Field + ": " + msg
			}
			if msg != "" {
				apiErr.Message = msg
			}
		}
	}

	if apiErr.Message == "" && len(body) > 0 {
		if isHTMLErrorBody(resp, body) {
			apiErr.Message = http.StatusText(resp.StatusCode)
		} else {
			apiErr.Message = strings.TrimSpace(string(body))
		}
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	return apiErr
}

func isHTMLErrorBody(resp *http.Response, body []byte) bool {
	if resp != nil {
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		if strings.Contains(ct, "text/html") {
			return true
		}
	}
	s := strings.ToLower(strings.TrimSpace(string(body)))
	if s == "" {
		return false
	}
	return strings.HasPrefix(s, "<!doctype html") || strings.HasPrefix(s, "<html") || strings.Contains(s, "<html")
}

func firstHeader(resp *http.Response, names ...string) string {
	for _, name := range names {
		if v := resp.Header.Get(name); v != "" {
			return v
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// FormatAPIError returns a stable diagnostic string.
func FormatAPIError(err error) string {
	if err == nil {
		return ""
	}
	if apiErr, ok := err.(*APIError); ok && apiErr != nil {
		return apiErr.Error()
	}
	return fmt.Sprintf("%v", err)
}
