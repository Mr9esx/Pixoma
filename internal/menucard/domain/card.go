package domain

// Card is a message the bot sends: media + text + inline buttons.
type Card struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Media   []Media      `json:"media,omitempty"`
	Text    string       `json:"text"`
	Buttons []CardButton `json:"buttons,omitempty"`
}

// CardButton is a button attached to a card message.
type CardButton struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Action Action `json:"action"`
}

// Media is an image/video/animation attached to a card.
type Media struct {
	Kind    string `json:"kind"` // image | video | animation
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}

func ValidateCard(c Card) error {
	if c.ID == "" {
		return ErrValidation
	}
	if c.Text == "" && len(c.Media) == 0 {
		return ErrValidation
	}
	for _, b := range c.Buttons {
		if b.Label == "" {
			return ErrValidation
		}
		if err := ValidateAction(b.Action); err != nil {
			return err
		}
	}
	return nil
}
