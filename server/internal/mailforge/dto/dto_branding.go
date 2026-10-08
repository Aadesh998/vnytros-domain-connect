package dto

type BrandingRequest struct {
	ShowWatermark     *bool  `json:"show_watermark"`
	WatermarkImageURL string `json:"watermark_image_url"`
	WatermarkLinkURL  string `json:"watermark_link_url"`
	WatermarkLabel    string `json:"watermark_label"`
}

type BrandingResponse struct {
	ShowWatermark     bool     `json:"show_watermark"`
	WatermarkImageURL string   `json:"watermark_image_url"`
	WatermarkLinkURL  string   `json:"watermark_link_url"`
	WatermarkLabel    string   `json:"watermark_label"`
	AllowedImageURLs  []string `json:"allowed_image_urls"`
}
