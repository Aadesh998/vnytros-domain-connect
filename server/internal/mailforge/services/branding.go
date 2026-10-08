package services

import (
	"context"
	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/authctx"
	"domain-connect-backend/internal/mailforge/config"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/repositary"
	"net/url"
	"slices"
	"strings"
)

// AllowedWatermarkImageURLs is the deployment's default watermark image plus
// MAIL_WATERMARK_ALLOWED_IMAGE_URLS. Empty entries are dropped, so a
// deployment that configures no images offers none.
func AllowedWatermarkImageURLs() []string {
	var out []string
	for _, u := range append([]string{config.AppConfig.WatermarkImageURL}, config.AppConfig.WatermarkAllowedImageURLs...) {
		if strings.TrimSpace(u) != "" {
			out = append(out, u)
		}
	}
	return out
}

func GetBranding(ctx context.Context) (dto.BrandingResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetBranding")
	defer span.End()

	resp := dto.BrandingResponse{
		ShowWatermark:     true,
		WatermarkImageURL: config.AppConfig.WatermarkImageURL,
		WatermarkLinkURL:  config.AppConfig.WatermarkLinkURL,
		WatermarkLabel:    config.AppConfig.WatermarkLabel,
		AllowedImageURLs:  dedupe(AllowedWatermarkImageURLs()),
	}

	branding, found, err := repositary.GetBranding(ctx, authctx.UserID(ctx))
	if err != nil {
		return dto.BrandingResponse{}, err
	}
	if !found {
		return resp, nil
	}

	resp.ShowWatermark = branding.ShowWatermark
	if branding.WatermarkImageURL != "" {
		resp.WatermarkImageURL = branding.WatermarkImageURL
	}
	if branding.WatermarkLinkURL != "" {
		resp.WatermarkLinkURL = branding.WatermarkLinkURL
	}
	if branding.WatermarkLabel != "" {
		resp.WatermarkLabel = branding.WatermarkLabel
	}

	return resp, nil
}

func UpdateBranding(ctx context.Context, req dto.BrandingRequest) (dto.BrandingResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "UpdateBranding")
	defer span.End()

	userID := authctx.UserID(ctx)

	branding, _, err := repositary.GetBranding(ctx, userID)
	if err != nil {
		return dto.BrandingResponse{}, err
	}
	branding.UserID = userID

	if req.ShowWatermark != nil {
		branding.ShowWatermark = *req.ShowWatermark
	}

	if img := strings.TrimSpace(req.WatermarkImageURL); img != "" {
		if !slices.Contains(AllowedWatermarkImageURLs(), img) {
			return dto.BrandingResponse{}, apperror.BadRequest.WithMessage(
				"That watermark image is not available. Choose one of the provided assets.")
		}
		branding.WatermarkImageURL = img
	}

	if link := strings.TrimSpace(req.WatermarkLinkURL); link != "" {
		if !isSafeHTTPURL(link) {
			return dto.BrandingResponse{}, apperror.BadRequest.WithMessage(
				"The watermark link must be an http(s) URL")
		}
		branding.WatermarkLinkURL = link
	}

	if label := strings.TrimSpace(req.WatermarkLabel); label != "" {
		if len(label) > 120 {
			label = label[:120]
		}
		branding.WatermarkLabel = label
	}

	if err := repositary.SaveBranding(ctx, &branding); err != nil {
		return dto.BrandingResponse{}, err
	}

	return GetBranding(ctx)
}

func ResolveWatermark(ctx context.Context) (jobs.Watermark, *apperror.AppError) {
	fallback := jobs.Watermark{
		Show:     true,
		ImageURL: config.AppConfig.WatermarkImageURL,
		LinkURL:  config.AppConfig.WatermarkLinkURL,
		Label:    config.AppConfig.WatermarkLabel,
	}

	branding, found, err := repositary.GetBranding(ctx, authctx.UserID(ctx))
	if err != nil {
		return jobs.Watermark{}, err
	}
	if !found {
		return fallback, nil
	}

	out := jobs.Watermark{
		Show:     branding.ShowWatermark,
		ImageURL: branding.WatermarkImageURL,
		LinkURL:  branding.WatermarkLinkURL,
		Label:    branding.WatermarkLabel,
	}
	if out.ImageURL == "" {
		out.ImageURL = fallback.ImageURL
	}
	if out.LinkURL == "" {
		out.LinkURL = fallback.LinkURL
	}
	if out.Label == "" {
		out.Label = fallback.Label
	}

	return out, nil
}

func isSafeHTTPURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
