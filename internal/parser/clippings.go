package parser

import (
	"strings"
	"time"
)

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

// Len ...
func (c Clippings) Len() int {
	return len(c)
}

// Less ...
func (c Clippings) Less(i, j int) bool {
	// First, sort alphabetically by the name of the Source
	if c[i].Source != c[j].Source {
		return strings.Compare(c[i].Source, c[j].Source) < 0
	}

	// Second, sort by the start location inside a given source
	if c[i].LocationInSource.Start != c[j].LocationInSource.Start {
		return c[i].LocationInSource.Start < c[j].LocationInSource.Start
	}

	// Third, sort by the type of clipping (i.e. all highlights with this start location will come
	// before the notes at this start location)
	if c[i].Type != c[j].Type {
		return c[i].Type < c[j].Type
	}

	// Fourth, sort reverse chronologically by create time
	return c[i].CreateTime.After(c[j].CreateTime)
}

// Swap ...
func (c Clippings) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
