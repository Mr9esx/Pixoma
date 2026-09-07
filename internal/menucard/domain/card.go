package domain

// Media is an image/video/animation attached to a card or button.
type Media struct {
	Kind    string `json:"kind"` // image | video | animation
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}
