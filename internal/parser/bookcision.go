package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
)

type BookcisionClippings struct {
	FilePath string
	Logger   *zap.Logger
}

type BookcisionClippingSet struct {
	Title      string `json:"title"`
	Authors    string `json:"authors"`
	Highlights []BookcisionHighlight
}

type BookcisionHighlight struct {
	Text       string
	IsNoteOnly bool
	Location   BookcisionHighlightLocation
	Note       string
}

type BookcisionHighlightLocation struct {
	URL   string
	Value int
}

// Parse ...
func (b *BookcisionClippings) Parse() (Clippings, error) {
	supplementFile, err := os.Open(b.FilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read input JSON file for supplements > %w", err)
	}
	defer supplementFile.Close()

	readerSupplement := json.NewDecoder(supplementFile)
	supplemented := BookcisionClippingSet{}
	if err := readerSupplement.Decode(&supplemented); err != nil {
		return nil, fmt.Errorf("could not encode clippings from bookcision JSON file > %w", err)
	}

	return b.convertToParser(supplemented)
}

// convertToParser ...
func (b *BookcisionClippings) convertToParser(in BookcisionClippingSet) (Clippings, error) {
	var output Clippings
	for _, hl := range in.Highlights {
		output = append(output, Clipping{
			Source: in.Title,
			Type:   ClippingType_Highlight,
			LocationInSource: Location{
				Start: hl.Location.Value,
			},
			Text: hl.Text,
		})

		if hl.Note != "" {
			output = append(output, Clipping{
				Source: in.Title,
				Type:   ClippingType_Note,
				LocationInSource: Location{
					Start: hl.Location.Value,
				},
				Text: hl.Note,
			})
		}
	}

	return output, nil
}
