// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// UserACL is one organization entry from GET /acls.
type UserACL struct {
	OrgID       int64    `json:"orgId"`
	OrgName     string   `json:"orgName"`
	DisplayName string   `json:"displayName"`
	RoleNames   []string `json:"roleNames"`
}

// OrgIDError is returned when the configured org_id is not in GET /acls.
type OrgIDError struct {
	Configured string
	Accessible []UserACL
	Cause      error
}

func (e *OrgIDError) Error() string {
	if e == nil {
		return "invalid Apple Ads org_id"
	}

	var b strings.Builder
	if e.Configured == "" {
		b.WriteString("org_id is empty.")
	} else {
		fmt.Fprintf(&b, "org_id %q was not returned by GET /acls.", e.Configured)
	}

	if len(e.Accessible) > 0 {
		labels := make([]string, 0, len(e.Accessible))
		for _, a := range e.Accessible {
			labels = append(labels, aclLabel(a))
		}
		fmt.Fprintf(&b, " Organizations for this API client: %s.", strings.Join(labels, ", "))
	}
	b.WriteString(" Set the org_id provider argument or APPLEADS_ORG_ID.")
	return b.String()
}

func (e *OrgIDError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func aclLabel(a UserACL) string {
	id := strconv.FormatInt(a.OrgID, 10)
	name := strings.TrimSpace(a.DisplayName)
	if name == "" {
		name = strings.TrimSpace(a.OrgName)
	}
	if name == "" {
		return id
	}
	return id + " (" + name + ")"
}

type skipErrorEnrichmentKey struct{}

func withoutErrorEnrichment(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipErrorEnrichmentKey{}, true)
}

func skipErrorEnrichment(ctx context.Context) bool {
	v, _ := ctx.Value(skipErrorEnrichmentKey{}).(bool)
	return v
}

// ListUserACLs returns organizations the authenticated API client can access.
func (c *Client) ListUserACLs(ctx context.Context) ([]UserACL, error) {
	return FetchAllPages[UserACL](withoutErrorEnrichment(ctx), c, http.MethodGet, "acls", 1000, nil)
}

// CheckOrgID reports whether orgID is present in the ACL list.
func CheckOrgID(orgID string, acls []UserACL) error {
	orgID = strings.TrimSpace(orgID)
	for _, a := range acls {
		if strconv.FormatInt(a.OrgID, 10) == orgID {
			return nil
		}
	}
	return &OrgIDError{Configured: orgID, Accessible: acls}
}

func (c *Client) enrichAPIError(ctx context.Context, apiErr *APIError) error {
	if apiErr == nil || c.orgID == "" || skipErrorEnrichment(ctx) {
		return apiErr
	}
	if apiErr.StatusCode != http.StatusForbidden {
		return apiErr
	}
	acls, err := c.ListUserACLs(ctx)
	if err != nil {
		return apiErr
	}
	if err := CheckOrgID(c.orgID, acls); err != nil {
		var orgErr *OrgIDError
		if errors.As(err, &orgErr) {
			orgErr.Cause = apiErr
			return orgErr
		}
		return err
	}
	return apiErr
}
