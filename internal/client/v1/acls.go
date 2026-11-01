// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// UserACL is one organization / access entry from GET /acls (Platform API).
// Preview notes: Me & ACL shapes change; orgId remains for org binding while
// ad accounts are discovered separately.
type UserACL struct {
	OrgID       int64    `json:"orgId"`
	OrgName     string   `json:"orgName"`
	DisplayName string   `json:"displayName"`
	RoleNames   []string `json:"roleNames"`
}

// AdAccount is an advertising account from Platform API ad-account endpoints.
type AdAccount struct {
	ID          int64  `json:"id"`
	OrgID       int64  `json:"orgId,omitempty"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Currency    string `json:"currency,omitempty"`
	TimeZone    string `json:"timeZone,omitempty"`
	Status      string `json:"status,omitempty"`
}

// AdAccountIDError is returned when the configured ad_account_id is invalid.
type AdAccountIDError struct {
	Configured string
	Accessible []AdAccount
	Cause      error
}

func (e *AdAccountIDError) Error() string {
	if e == nil {
		return "invalid Apple Ads ad_account_id"
	}
	var b strings.Builder
	if e.Configured == "" {
		b.WriteString("ad_account_id is empty.")
	} else {
		fmt.Fprintf(&b, "ad_account_id %q was not found among accessible ad accounts.", e.Configured)
	}
	if len(e.Accessible) > 0 {
		labels := make([]string, 0, len(e.Accessible))
		for _, a := range e.Accessible {
			labels = append(labels, adAccountLabel(a))
		}
		fmt.Fprintf(&b, " Accessible ad accounts for this API client: %s.", strings.Join(labels, ", "))
	}
	b.WriteString(" Set the ad_account_id provider argument or APPLEADS_AD_ACCOUNT_ID. Do not reuse org_id.")
	return b.String()
}

func (e *AdAccountIDError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func adAccountLabel(a AdAccount) string {
	id := strconv.FormatInt(a.ID, 10)
	name := strings.TrimSpace(a.DisplayName)
	if name == "" {
		name = strings.TrimSpace(a.Name)
	}
	if name == "" {
		return id
	}
	return id + " (" + name + ")"
}

// ListUserACLs returns ACL entries for the authenticated API client (unscoped).
func (c *Client) ListUserACLs(ctx context.Context) ([]UserACL, error) {
	var env ListResponse[UserACL]
	if err := c.DoJSON(ctx, http.MethodGet, "acls", nil, &env); err != nil {
		return nil, err
	}
	return env.Result, nil
}

// ListAdAccounts returns ad accounts visible to the API client (unscoped).
func (c *Client) ListAdAccounts(ctx context.Context) ([]AdAccount, error) {
	// Prefer /query when available; fall back to GET /adaccounts.
	if page, err := QueryPage[AdAccount](ctx, c, "adaccounts/query", QueryRequest{}); err == nil {
		return page.Data, nil
	}
	var env ListResponse[AdAccount]
	if err := c.DoJSON(ctx, http.MethodGet, "adaccounts", nil, &env); err != nil {
		return nil, err
	}
	return env.Result, nil
}

// GetAdAccount fetches one ad account by ID.
func (c *Client) GetAdAccount(ctx context.Context, id int64) (*AdAccount, error) {
	var env Response[AdAccount]
	path := fmt.Sprintf("adaccounts/%d", id)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	return &env.Result, nil
}

// CheckAdAccountID reports whether adAccountID is present in the account list.
func CheckAdAccountID(adAccountID string, accounts []AdAccount) error {
	adAccountID = strings.TrimSpace(adAccountID)
	for _, a := range accounts {
		if strconv.FormatInt(a.ID, 10) == adAccountID {
			return nil
		}
	}
	return &AdAccountIDError{Configured: adAccountID, Accessible: accounts}
}

// ValidateAdAccountID lists ad accounts and ensures the configured ID is present.
func (c *Client) ValidateAdAccountID(ctx context.Context) error {
	if strings.TrimSpace(c.adAccountID) == "" {
		return &AdAccountIDError{Configured: ""}
	}
	accounts, err := c.ListAdAccounts(ctx)
	if err != nil {
		return err
	}
	return CheckAdAccountID(c.adAccountID, accounts)
}
