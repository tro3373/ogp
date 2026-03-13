package ogp

// Result holds the extracted OGP metadata for a URL.
type Result struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Err         error  `json:"-"`
}
