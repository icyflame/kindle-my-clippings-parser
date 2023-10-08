package duplicates

import (
	"sort"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"go.uber.org/zap"
)

type Remover interface {
	Delete(parser.Clippings) (parser.Clippings, error)
}

type RetainLatest struct {
	Logger *zap.Logger
}

// Delete ...
func (r *RetainLatest) Delete(input parser.Clippings) (parser.Clippings, error) {
	output := make(parser.Clippings, 0)
	sort.Sort(input)
	for _, clipping := range input {
		// Always copy the first thing from input to output
		if len(output) == 0 {
			output = append(output, clipping)
			continue
		}

		// This is not a note. We are not deduplicating non-Notes for the time being. So, append
		// as-is.
		if clipping.Type != parser.ClippingType_Note {
			output = append(output, clipping)
			continue
		}

		// For notes: if the source, location start, and type (implicit) are the same, then retain
		// only the first note we see at a given source and location start.
		//
		// The logic for ordering clippings in this order is inside the sort.Interface function
		// impelemntations of the parser.Clippings type
		previous := output[len(output)-1]
		if clipping.Source == previous.Source &&
			clipping.LocationInSource.Start == previous.LocationInSource.Start &&
			clipping.Type == previous.Type {
			// Current clipping "clipping" is a duplicate of the previous clipping "previous"
			r.Logger.Debug("duplicate found", zap.Any("original", previous), zap.Any("current", clipping))
			continue
		}

		output = append(output, clipping)
	}

	return output, nil
}
