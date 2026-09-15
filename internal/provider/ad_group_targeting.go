// Copyright (c) 2026 Pi Cubed LLC
// SPDX-License-Identifier: MIT

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/picubedllc/terraform-provider-appleads/internal/client"
)

type targetingDimensionsModel struct {
	Age            *ageTargetModel            `tfsdk:"age"`
	Gender         *includedStringsModel      `tfsdk:"gender"`
	DeviceClass    *includedStringsModel      `tfsdk:"device_class"`
	Country        *includedStringsModel      `tfsdk:"country"`
	AdminArea      *includedStringsModel      `tfsdk:"admin_area"`
	Locality       *includedStringsModel      `tfsdk:"locality"`
	Daypart        *daypartTargetModel        `tfsdk:"daypart"`
	AppDownloaders *appDownloadersTargetModel `tfsdk:"app_downloaders"`
}

type includedStringsModel struct {
	Included types.List `tfsdk:"included"`
}

type ageTargetModel struct {
	Included []ageRangeModel `tfsdk:"included"`
}

type ageRangeModel struct {
	MinAge types.Int64 `tfsdk:"min_age"`
	MaxAge types.Int64 `tfsdk:"max_age"`
}

type daypartTargetModel struct {
	UserTime *includedInt64sModel `tfsdk:"user_time"`
}

type includedInt64sModel struct {
	Included types.List `tfsdk:"included"`
}

type appDownloadersTargetModel struct {
	Included types.List `tfsdk:"included"`
	Excluded types.List `tfsdk:"excluded"`
}

func targetingDimensionsAttribute() schema.SingleNestedAttribute {
	includedStrings := func(desc string) schema.SingleNestedAttribute {
		return schema.SingleNestedAttribute{
			Optional:            true,
			MarkdownDescription: desc,
			Attributes: map[string]schema.Attribute{
				"included": schema.ListAttribute{
					Required:            true,
					ElementType:         types.StringType,
					MarkdownDescription: "Apple criteria IDs to include.",
					Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
				},
			},
		}
	}

	return schema.SingleNestedAttribute{
		Optional: true,
		MarkdownDescription: "Audience targeting mapped to Apple Ads `targetingDimensions` (mutable).\n\n" +
			"Geo targeting (`country`, `admin_area`, `locality`) only works on **single-country** campaigns. " +
			"Campaign `countries_or_regions` stays at country grain (immutable after create); city grain is this ad-group object. " +
			"Look up IDs with the `appleads_geolocations` data source. Examples: country `US` (ISO alpha-2), " +
			"admin area `US|NY`, locality `US|NY|New York`.\n\n" +
			"Omitted nested dimensions are left unset on create. On update, Apple requires a full `targetingDimensions` object: " +
			"this provider sends JSON `null` for omitted nested dimensions so they clear. Removing `targeting_dimensions` " +
			"entirely after it was set sends `targetingDimensions: null`. Apple may still return default `deviceClass` " +
			"for the promoted app; unmanaged nested dimensions are not copied into state.",
		Attributes: map[string]schema.Attribute{
			"locality": includedStrings(
				"City targeting. IDs are `Country|AdminArea|Locality`, e.g. `US|NY|New York`. Resolve with `appleads_geolocations` (`entity = \"Locality\"`).",
			),
			"admin_area": includedStrings(
				"State/province targeting. IDs are `Country|AdminArea`, e.g. `US|NY`. Resolve with `appleads_geolocations` (`entity = \"AdminArea\"`).",
			),
			"country": includedStrings(
				"Country targeting using ISO alpha-2 codes, e.g. `US`. Geo targeting is not supported on multi-country campaigns.",
			),
			"age": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Age ranges to include. Minimum age is 18. `max_age` may be omitted (no upper bound).",
				Attributes: map[string]schema.Attribute{
					"included": schema.ListNestedAttribute{
						Required:            true,
						MarkdownDescription: "One or more age ranges.",
						Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"min_age": schema.Int64Attribute{
									Required:            true,
									MarkdownDescription: "Minimum age (18 or older).",
									Validators:          []validator.Int64{int64validator.AtLeast(18)},
								},
								"max_age": schema.Int64Attribute{
									Optional:            true,
									MarkdownDescription: "Optional maximum age. Omit for no upper bound.",
								},
							},
						},
					},
				},
			},
			"gender": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Gender targeting. Apple Ads values are `M` and `F`.",
				Attributes: map[string]schema.Attribute{
					"included": schema.ListAttribute{
						Required:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "One or more of `M`, `F`.",
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.ValueStringsAre(stringvalidator.OneOf("M", "F")),
						},
					},
				},
			},
			"device_class": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Device class targeting. Values are `IPHONE` and `IPAD`.",
				Attributes: map[string]schema.Attribute{
					"included": schema.ListAttribute{
						Required:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "One or more of `IPHONE`, `IPAD`.",
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
							listvalidator.ValueStringsAre(stringvalidator.OneOf("IPHONE", "IPAD")),
						},
					},
				},
			},
			"daypart": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Limit serving to hours of the week in the user's time zone. Hours are 0–167 starting Sunday 12:00 AM (Monday 1:00 AM is 25).",
				Attributes: map[string]schema.Attribute{
					"user_time": schema.SingleNestedAttribute{
						Required:            true,
						MarkdownDescription: "User-local hours of the week to include.",
						Attributes: map[string]schema.Attribute{
							"included": schema.ListAttribute{
								Required:            true,
								ElementType:         types.Int64Type,
								MarkdownDescription: "Hours of the week (0–167).",
								Validators: []validator.List{
									listvalidator.SizeAtLeast(1),
									listvalidator.ValueInt64sAre(int64validator.Between(0, 167)),
								},
							},
						},
					},
				},
			},
			"app_downloaders": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Target users who have or have not downloaded apps you own. Values are Adam IDs as strings. `excluded` is typically the campaign's promoted app.",
				Attributes: map[string]schema.Attribute{
					"included": schema.ListAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Adam IDs of owned apps whose downloaders to include.",
					},
					"excluded": schema.ListAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Adam IDs whose downloaders to exclude (usually the promoted app).",
					},
				},
			},
		},
	}
}

func targetingDimensionsFromModel(ctx context.Context, m *targetingDimensionsModel) (*client.TargetingDimensions, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m == nil {
		return nil, diags
	}
	out := &client.TargetingDimensions{}
	out.Age = ageTargetFromModel(m.Age)
	gender, d := genderTargetFromModel(ctx, m.Gender)
	diags.Append(d...)
	out.Gender = gender
	device, d := deviceClassTargetFromModel(ctx, m.DeviceClass)
	diags.Append(d...)
	out.DeviceClass = device
	country, d := localityTargetFromModel(ctx, m.Country)
	diags.Append(d...)
	out.Country = country
	admin, d := localityTargetFromModel(ctx, m.AdminArea)
	diags.Append(d...)
	out.AdminArea = admin
	locality, d := localityTargetFromModel(ctx, m.Locality)
	diags.Append(d...)
	out.Locality = locality
	daypart, d := daypartTargetFromModel(ctx, m.Daypart)
	diags.Append(d...)
	out.Daypart = daypart
	downloaders, d := appDownloadersFromModel(ctx, m.AppDownloaders)
	diags.Append(d...)
	out.AppDownloaders = downloaders
	if diags.HasError() {
		return nil, diags
	}
	return out, diags
}

func localityTargetFromModel(ctx context.Context, m *includedStringsModel) (*client.LocalityTarget, diag.Diagnostics) {
	included, diags := includedStringsFromModel(ctx, m)
	if included == nil {
		return nil, diags
	}
	return &client.LocalityTarget{Included: included}, diags
}

func genderTargetFromModel(ctx context.Context, m *includedStringsModel) (*client.GenderTarget, diag.Diagnostics) {
	included, diags := includedStringsFromModel(ctx, m)
	if included == nil {
		return nil, diags
	}
	return &client.GenderTarget{Included: included}, diags
}

func deviceClassTargetFromModel(ctx context.Context, m *includedStringsModel) (*client.DeviceClassTarget, diag.Diagnostics) {
	included, diags := includedStringsFromModel(ctx, m)
	if included == nil {
		return nil, diags
	}
	return &client.DeviceClassTarget{Included: included}, diags
}

func includedStringsFromModel(ctx context.Context, m *includedStringsModel) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m == nil {
		return nil, diags
	}
	included, d := stringList(ctx, m.Included)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	return included, diags
}

func ageTargetFromModel(m *ageTargetModel) *client.AgeTarget {
	if m == nil {
		return nil
	}
	included := make([]client.AgeRange, 0, len(m.Included))
	for _, r := range m.Included {
		ar := client.AgeRange{MinAge: int(r.MinAge.ValueInt64())}
		if !r.MaxAge.IsNull() && !r.MaxAge.IsUnknown() {
			ar.MaxAge = int(r.MaxAge.ValueInt64())
		}
		included = append(included, ar)
	}
	return &client.AgeTarget{Included: included}
}

func daypartTargetFromModel(ctx context.Context, m *daypartTargetModel) (*client.DaypartTarget, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m == nil || m.UserTime == nil {
		return nil, diags
	}
	hours, d := int64List(ctx, m.UserTime.Included)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	included := make([]int, 0, len(hours))
	for _, h := range hours {
		included = append(included, int(h))
	}
	return &client.DaypartTarget{UserTime: &client.DaypartUserTime{Included: included}}, diags
}

func appDownloadersFromModel(ctx context.Context, m *appDownloadersTargetModel) (*client.AppDownloadersTarget, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m == nil {
		return nil, diags
	}
	included, d := stringList(ctx, m.Included)
	diags.Append(d...)
	excluded, d := stringList(ctx, m.Excluded)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	return &client.AppDownloadersTarget{Included: included, Excluded: excluded}, diags
}

func targetingDimensionsToModel(ctx context.Context, t *client.TargetingDimensions) (*targetingDimensionsModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if t == nil || targetingDimensionsEmpty(t) {
		return nil, diags
	}
	out := &targetingDimensionsModel{}
	out.Age = ageTargetToModel(t.Age)
	gender, d := includedStringsToModel(ctx, genderIncluded(t.Gender))
	diags.Append(d...)
	out.Gender = gender
	device, d := includedStringsToModel(ctx, deviceIncluded(t.DeviceClass))
	diags.Append(d...)
	out.DeviceClass = device
	country, d := includedStringsToModel(ctx, localityIncluded(t.Country))
	diags.Append(d...)
	out.Country = country
	admin, d := includedStringsToModel(ctx, localityIncluded(t.AdminArea))
	diags.Append(d...)
	out.AdminArea = admin
	locality, d := includedStringsToModel(ctx, localityIncluded(t.Locality))
	diags.Append(d...)
	out.Locality = locality
	daypart, d := daypartTargetToModel(ctx, t.Daypart)
	diags.Append(d...)
	out.Daypart = daypart
	downloaders, d := appDownloadersToModel(ctx, t.AppDownloaders)
	diags.Append(d...)
	out.AppDownloaders = downloaders
	if targetingDimensionsModelEmpty(out) {
		return nil, diags
	}
	return out, diags
}

func targetingDimensionsEmpty(t *client.TargetingDimensions) bool {
	if t == nil {
		return true
	}
	return t.Age == nil && t.Gender == nil && t.DeviceClass == nil && t.Country == nil &&
		t.AdminArea == nil && t.Locality == nil && t.Daypart == nil && appDownloadersEmpty(t.AppDownloaders)
}

func appDownloadersEmpty(t *client.AppDownloadersTarget) bool {
	return t == nil || (len(t.Included) == 0 && len(t.Excluded) == 0)
}

func targetingDimensionsModelEmpty(m *targetingDimensionsModel) bool {
	if m == nil {
		return true
	}
	return m.Age == nil && m.Gender == nil && m.DeviceClass == nil && m.Country == nil &&
		m.AdminArea == nil && m.Locality == nil && m.Daypart == nil && m.AppDownloaders == nil
}

func genderIncluded(t *client.GenderTarget) []string {
	if t == nil {
		return nil
	}
	return t.Included
}

func deviceIncluded(t *client.DeviceClassTarget) []string {
	if t == nil {
		return nil
	}
	return t.Included
}

func localityIncluded(t *client.LocalityTarget) []string {
	if t == nil {
		return nil
	}
	return t.Included
}

func includedStringsToModel(ctx context.Context, included []string) (*includedStringsModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(included) == 0 {
		return nil, diags
	}
	list, d := types.ListValueFrom(ctx, types.StringType, included)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	return &includedStringsModel{Included: list}, diags
}

func ageTargetToModel(t *client.AgeTarget) *ageTargetModel {
	if t == nil || len(t.Included) == 0 {
		return nil
	}
	included := make([]ageRangeModel, 0, len(t.Included))
	for _, r := range t.Included {
		item := ageRangeModel{MinAge: types.Int64Value(int64(r.MinAge))}
		if r.MaxAge > 0 {
			item.MaxAge = types.Int64Value(int64(r.MaxAge))
		} else {
			item.MaxAge = types.Int64Null()
		}
		included = append(included, item)
	}
	return &ageTargetModel{Included: included}
}

func daypartTargetToModel(ctx context.Context, t *client.DaypartTarget) (*daypartTargetModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if t == nil || t.UserTime == nil || len(t.UserTime.Included) == 0 {
		return nil, diags
	}
	hours := make([]int64, 0, len(t.UserTime.Included))
	for _, h := range t.UserTime.Included {
		hours = append(hours, int64(h))
	}
	list, d := types.ListValueFrom(ctx, types.Int64Type, hours)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	return &daypartTargetModel{UserTime: &includedInt64sModel{Included: list}}, diags
}

func appDownloadersToModel(ctx context.Context, t *client.AppDownloadersTarget) (*appDownloadersTargetModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if appDownloadersEmpty(t) {
		return nil, diags
	}
	out := &appDownloadersTargetModel{
		Included: types.ListNull(types.StringType),
		Excluded: types.ListNull(types.StringType),
	}
	if len(t.Included) > 0 {
		list, d := types.ListValueFrom(ctx, types.StringType, t.Included)
		diags.Append(d...)
		out.Included = list
	}
	if len(t.Excluded) > 0 {
		list, d := types.ListValueFrom(ctx, types.StringType, t.Excluded)
		diags.Append(d...)
		out.Excluded = list
	}
	if diags.HasError() {
		return nil, diags
	}
	return out, diags
}

// overlayAdGroupTargeting keeps configured targeting dimensions and list order.
// Apple returns default deviceClass (and empty appDownloaders) when those dimensions
// were never set; unmanaged nested fields stay null in state so Create does not
// fail Terraform's after-apply consistency check.
func overlayAdGroupTargeting(ctx context.Context, configured, reported adGroupModel) (adGroupModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if configured.TargetingDimensions == nil {
		reported.TargetingDimensions = nil
		return reported, diags
	}
	if reported.TargetingDimensions == nil {
		reported.TargetingDimensions = configured.TargetingDimensions
		return reported, diags
	}
	cfg := configured.TargetingDimensions
	rep := reported.TargetingDimensions
	out := &targetingDimensionsModel{}
	if cfg.Age != nil {
		if rep.Age != nil {
			out.Age = rep.Age
		} else {
			out.Age = cfg.Age
		}
	}
	loc, d := overlayIncludedStrings(ctx, cfg.Gender, rep.Gender)
	diags.Append(d...)
	out.Gender = loc
	loc, d = overlayIncludedStrings(ctx, cfg.DeviceClass, rep.DeviceClass)
	diags.Append(d...)
	out.DeviceClass = loc
	loc, d = overlayIncludedStrings(ctx, cfg.Country, rep.Country)
	diags.Append(d...)
	out.Country = loc
	loc, d = overlayIncludedStrings(ctx, cfg.AdminArea, rep.AdminArea)
	diags.Append(d...)
	out.AdminArea = loc
	loc, d = overlayIncludedStrings(ctx, cfg.Locality, rep.Locality)
	diags.Append(d...)
	out.Locality = loc
	if cfg.Daypart != nil {
		if rep.Daypart != nil {
			hours, d := overlayInt64List(ctx, cfg.Daypart.UserTime, rep.Daypart.UserTime)
			diags.Append(d...)
			out.Daypart = &daypartTargetModel{UserTime: hours}
		} else {
			out.Daypart = cfg.Daypart
		}
	}
	dl, d := overlayAppDownloaders(ctx, cfg.AppDownloaders, rep.AppDownloaders)
	diags.Append(d...)
	out.AppDownloaders = dl
	reported.TargetingDimensions = out
	return reported, diags
}

func overlayIncludedStrings(ctx context.Context, configured, reported *includedStringsModel) (*includedStringsModel, diag.Diagnostics) {
	if configured == nil {
		return nil, nil
	}
	if reported == nil {
		return configured, nil
	}
	list, d := preferConfiguredStringList(ctx, configured.Included, reported.Included)
	return &includedStringsModel{Included: list}, d
}

func overlayInt64List(ctx context.Context, configured, reported *includedInt64sModel) (*includedInt64sModel, diag.Diagnostics) {
	if configured == nil {
		return nil, nil
	}
	if reported == nil {
		return configured, nil
	}
	cfg, d := int64List(ctx, configured.Included)
	if d.HasError() {
		return reported, d
	}
	rep, d2 := int64List(ctx, reported.Included)
	d.Append(d2...)
	if d.HasError() {
		return reported, d
	}
	if int64SetEqual(cfg, rep) {
		return configured, d
	}
	return reported, d
}

func overlayAppDownloaders(ctx context.Context, configured, reported *appDownloadersTargetModel) (*appDownloadersTargetModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if configured == nil {
		return nil, diags
	}
	if reported == nil {
		return configured, diags
	}
	included, d := overlayOptionalStringList(ctx, configured.Included, reported.Included)
	diags.Append(d...)
	excluded, d := overlayOptionalStringList(ctx, configured.Excluded, reported.Excluded)
	diags.Append(d...)
	return &appDownloadersTargetModel{Included: included, Excluded: excluded}, diags
}

func overlayOptionalStringList(ctx context.Context, configured, reported types.List) (types.List, diag.Diagnostics) {
	if configured.IsNull() || configured.IsUnknown() {
		return types.ListNull(types.StringType), nil
	}
	return preferConfiguredStringList(ctx, configured, reported)
}

func int64SetEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[int64]int, len(a))
	for _, n := range a {
		counts[n]++
	}
	for _, n := range b {
		counts[n]--
		if counts[n] < 0 {
			return false
		}
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}

func int64List(ctx context.Context, list types.List) ([]int64, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var out []int64
	diags.Append(list.ElementsAs(ctx, &out, false)...)
	return out, diags
}
