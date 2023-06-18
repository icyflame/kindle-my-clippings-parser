package parser

import "time"

type Location struct {
	Start int `yaml:",omitempty"`
	End   int `yaml:",omitempty"`
}

type ClippingType int

const (
	ClippingType_None ClippingType = iota
	ClippingType_Highlight
	ClippingType_Note
)

type Clipping struct {
	Source string       `yaml:"source"`
	Type   ClippingType `yaml:"type"`

	// Page is not always a number. Sometimes it is a lowercase Roman numeral "ix"
	Page string `yaml:"page"`

	LocationInSource Location  `yaml:"location_in_source"`
	CreateTime       time.Time `yaml:"create_time"`
	Text             string    `yaml:"text"`
}

type Clippings []Clipping
