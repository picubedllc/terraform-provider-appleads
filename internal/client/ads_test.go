// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAd_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/campaigns/10/adgroups/20/ads" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body AdCreate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "Summer Ad" || body.CreativeID != 94895512 || body.Status != AdStatusPaused {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":           501,
				"campaignId":   10,
				"adGroupId":    20,
				"name":         body.Name,
				"creativeId":   body.CreativeID,
				"creativeType": "CUSTOM_PRODUCT_PAGE",
				"status":       body.Status,
				"deleted":      false,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.CreateAd(context.Background(), 10, 20, &AdCreate{
		Name: "Summer Ad", CreativeID: 94895512, Status: AdStatusPaused,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 501 || out.CreativeID != 94895512 || out.CreativeType != "CUSTOM_PRODUCT_PAGE" {
		t.Fatalf("out = %#v", out)
	}
}

func TestCreateAd_Validation(t *testing.T) {
	t.Parallel()

	c, err := New(WithBaseURL("http://example.invalid"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateAd(context.Background(), 1, 2, &AdCreate{Name: "x", CreativeID: 0, Status: AdStatusEnabled})
	if err == nil || !strings.Contains(err.Error(), "creativeId") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetAd_SoftDeleted(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/campaigns/1/adgroups/2/ads/3" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id": 3, "campaignId": 1, "adGroupId": 2, "name": "gone", "deleted": true,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.GetAd(context.Background(), 1, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Deleted {
		t.Fatalf("out = %#v", out)
	}
}

func TestUpdateAd_NameAndStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/campaigns/10/adgroups/20/ads/501" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body AdUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "Renamed" || body.Status != AdStatusEnabled || body.CreativeID != 99 {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id": 501, "name": body.Name, "status": body.Status, "creativeId": body.CreativeID,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.UpdateAd(context.Background(), 10, 20, 501, &AdUpdate{
		Name: "Renamed", Status: AdStatusEnabled, CreativeID: 99,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Renamed" || out.Status != AdStatusEnabled || out.CreativeID != 99 {
		t.Fatalf("out = %#v", out)
	}
}

func TestDeleteAd_Success(t *testing.T) {
	t.Parallel()

	deleted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/campaigns/10/adgroups/20/ads/501" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		deleted = true
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteAd(context.Background(), 10, 20, 501); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected DELETE")
	}
}

func TestFindAdsInCampaign_AndOrg(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/campaigns/10/ads/find":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 501, "campaignId": 10, "adGroupId": 20}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/ads/find":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 777, "campaignId": 1, "adGroupId": 2}},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	inCampaign, err := c.FindAdsInCampaign(context.Background(), 10, Selector{})
	if err != nil || len(inCampaign) != 1 || inCampaign[0].ID != 501 {
		t.Fatalf("campaign find = %#v err=%v", inCampaign, err)
	}
	org, err := c.FindAds(context.Background(), Selector{})
	if err != nil || len(org) != 1 || org[0].ID != 777 {
		t.Fatalf("org find = %#v err=%v", org, err)
	}
}

func TestFindAdByID_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.FindAdByID(context.Background(), 10, 501)
	if !IsNotFound(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateAd_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"errors": []map[string]string{{
					"messageCode": "INVALID_ATTRIBUTE_VALUE",
					"message":     "creativeId is invalid",
					"field":       "creativeId",
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.CreateAd(context.Background(), 1, 2, &AdCreate{
		Name: "x", CreativeID: 9, Status: AdStatusPaused,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "INVALID_ATTRIBUTE_VALUE" {
		t.Fatalf("err = %#v", err)
	}
}

func TestParseAdImportID(t *testing.T) {
	t.Parallel()

	cID, agID, adID, err := ParseAdImportID("10/20/501")
	if err != nil || cID != 10 || agID != 20 || adID != 501 {
		t.Fatalf("%d/%d/%d err=%v", cID, agID, adID, err)
	}
	cID, agID, adID, err = ParseAdImportID("20/501")
	if err != nil || cID != 0 || agID != 20 || adID != 501 {
		t.Fatalf("%d/%d/%d err=%v", cID, agID, adID, err)
	}
	if _, _, _, err := ParseAdImportID("bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestListAds_Path(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/campaigns/10/adgroups/20/ads" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": 1}},
			"pagination": map[string]int{
				"totalResults": 1, "startIndex": 0, "itemsPerPage": 1,
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := New(WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.ListAds(context.Background(), 10, 20)
	if err != nil || len(out) != 1 || out[0].ID != 1 {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}
