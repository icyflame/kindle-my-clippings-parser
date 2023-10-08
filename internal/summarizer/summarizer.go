package summarizer

import (
	"errors"
	"strconv"
	"strings"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"go.uber.org/zap"
)

type BookSummary struct {
	Name     string
	Chapters []ChapterSummary
}

type ChapterSummary struct {
	Name             string
	Level            string
	SummaryClippings parser.Clippings
}

type Creator interface {
	Summarize(parser.Clippings) (BookSummary, error)
}

type KindleCreator struct {
	Logger *zap.Logger
}

// Summarize ...
func (k *KindleCreator) Summarize(input parser.Clippings) (BookSummary, error) {
	if len(input) == 0 {
		return BookSummary{}, errors.New("no clippings to summarize")
	}
	summ := BookSummary{}
	summ.Name = input[0].Source
	for i, clipping := range input {
		// Chapter name prefix
		if strings.HasPrefix(clipping.Text, "#cn") {
			// Get the chapter name from the previous highlight
			chapterName := input[i-1].Text
			thisChapter := ChapterSummary{
				Name:             chapterName,
				Level:            "*",
				SummaryClippings: []parser.Clipping{},
			}

			if parsedLevel, hasLevel := strings.CutPrefix(clipping.Text, "#cn "); hasLevel {
				if level, err := strconv.Atoi(parsedLevel); err == nil {
					thisChapter.Level = strings.Repeat("*", level)
				}
			}

			summ.Chapters = append(summ.Chapters, thisChapter)
		}
	}

	return summ, nil
}
