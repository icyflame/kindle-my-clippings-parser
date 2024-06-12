package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"
)

type KindleClippings struct {
	FilePath                     string
	RemoveClippingLimitClippings bool
	logger                       *zap.Logger
}

type LineType int

const (
	LineType_Invalid LineType = iota
	LineType_Source
	LineType_Description
	LineType_Empty
	LineType_Clipping
)

const KindleClippingLimitMessage = "<You have reached the clipping limit for this item>"

type Variant int

const (
	Variant_English_BooksWithPageNum Variant = iota
	Variant_English_BooksWithoutPageNum
	Variant_Japanese
)

const KindleClippingsSeparator = "=========="

var KindleClippingsSeparatorMatcher *regexp.Regexp = regexp.MustCompile(`={10}`)

type KindleDescriptionLineVariation struct {
	Matcher               *regexp.Regexp
	RequiredMatchCount    int
	Type                  [2]int
	Page                  [2]int
	LocationInSourceStart [2]int
	LocationInSourceEnd   [2]int
	CreateTime            [2]int
	CreateTimeFormat      string
}

var KindleDescriptionLineVariations []KindleDescriptionLineVariation = []KindleDescriptionLineVariation{
	// Sample line: Variation 1: Books with page numbers
	// "- Your Highlight on page 373 | location 5709-5720 | Added on Sunday, 16 April 2023 10:13:54"
	// "- Your Note on page 286 | location 4371 | Added on Saturday, 15 April 2023 12:51:43"
	{
		Matcher:               regexp.MustCompile(`^- Your (.+?) on page ([ivx0-9]+) \| location (\d+)-?(\d+)? \| Added on (.+)$`),
		RequiredMatchCount:    12,
		Type:                  [2]int{2, 3},
		Page:                  [2]int{4, 5},
		LocationInSourceStart: [2]int{6, 7},
		LocationInSourceEnd:   [2]int{8, 9},
		CreateTime:            [2]int{10, 11},
		CreateTimeFormat:      "Monday, 2 January 2006 15:04:05",
	},
	// Sample line: Variation 2: Books with out page numbers
	// "- Your Highlight at location 9723-9727 | Added on Sunday, 2 January 2022 13:17:22"
	// "- Your Note at location 9727 | Added on Sunday, 2 January 2022 13:17:46"
	{
		Matcher:               regexp.MustCompile(`^- Your (.+?) at location (\d+)-?(\d+)? \| Added on (.+)$`),
		RequiredMatchCount:    10,
		Type:                  [2]int{2, 3},
		Page:                  [2]int{-1, -1},
		LocationInSourceStart: [2]int{4, 5},
		LocationInSourceEnd:   [2]int{6, 7},
		CreateTime:            [2]int{8, 9},
		CreateTimeFormat:      "Monday, 2 January 2006 15:04:05",
	},
	// Sample line: Variation 3: Japanese
	// "- 22ページ|位置No. 336のメモ |作成日: 2023年6月10日土曜日 9:18:40"
	// "- 7ページ|位置No. 96-96のハイライト |作成日: 2023年5月14日日曜日 11:31:52"
	// "- 1ページ|位置No. 5-5のハイライト |作成日: 2023年5月13日土曜日 19:47:14"
	{
		Matcher:               regexp.MustCompile(`- (\d+)ページ\|位置No. (\d+)-?(\d+)?の(.+) \|作成日: (.+)`),
		RequiredMatchCount:    12,
		Type:                  [2]int{8, 9},
		Page:                  [2]int{2, 3},
		LocationInSourceStart: [2]int{4, 5},
		LocationInSourceEnd:   [2]int{6, 7},
		CreateTime:            [2]int{10, 11},
		CreateTimeFormat:      "2006年1月2日",
	},
	// Sample line - Variation 4 - English (since June 2023)
	// Probably because of a Kindle software update
	//
	// - Your Highlight on page 4 | Location 52-54 | Added on Wednesday, June 14, 2023 10:34:06 PM
	{
		Matcher:               regexp.MustCompile(`^- Your (.+?) on page ([ivx0-9]+) \| Location (\d+)-?(\d+)? \| Added on (.+)$`),
		RequiredMatchCount:    12,
		Type:                  [2]int{2, 3},
		Page:                  [2]int{4, 5},
		LocationInSourceStart: [2]int{6, 7},
		LocationInSourceEnd:   [2]int{8, 9},
		CreateTime:            [2]int{10, 11},
		CreateTimeFormat:      "Monday, January 2, 2006 3:04:05 PM",
	},
	// Sample Line - Variation 5 - English (since June 2024)
	// Probably because of a Kindle software update
	// - Your Highlight on Location 136-138 | Added on Tuesday, March 19, 2024 9:45:15 PM
	{
		Matcher:               regexp.MustCompile(`^- Your (.+?) on Location (\d+)-?(\d+)? \| Added on (.+)$`),
		RequiredMatchCount:    10,
		Type:                  [2]int{2, 3},
		Page:                  [2]int{-1, -1},
		LocationInSourceStart: [2]int{4, 5},
		LocationInSourceEnd:   [2]int{6, 7},
		CreateTime:            [2]int{8, 9},
		CreateTimeFormat:      "Monday, January 2, 2006 3:04:05 PM",
	},
}

// A sample clipping:
//
// --- Sample START ---
//
// Alias Grace (Atwood, Margaret)
// - Your Highlight on page 22 | location 281-283 | Added on Sunday, 5 May 2019 10:23:20
//
// They were bell-shaped and ruffled, gracefully waving and lovely under the sea; but if they washed up on the beach and dried out in the sun there was nothing left of them. And that is what the ladies are like: mostly water.
// ==========
//
// --- Sample END ---
//
// An alternate form of the second line is:
//
// --- Sample START ---
//
// - Your Highlight on page ix | location 341-344 | Added on Saturday, 25 January 2020 10:47:54
//
// --- Sample END ---
func (k *KindleClippings) Parse() (Clippings, error) {
	clippings, err := os.Open(k.FilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open the clippings file > %w", err)
	}

	defer clippings.Close()

	var output []Clipping

	scanner := bufio.NewScanner(clippings)
	scanner.Split(k.scanUsingKindleClippingsSeparator)

	// Initial character of Kindle's file is also a separator. So, we have to scan at least once to
	// get rid of that section.
	scanner.Scan()

	for scanner.Scan() {
		lineContent := scanner.Bytes()
		lineContent = bytes.TrimSpace(lineContent)
		components := bytes.SplitN(lineContent, []byte{'\n'}, 4)

		if k.isException(components) {
			continue
		}

		if len(lineContent) == 0 {
			continue
		}

		if len(components) != 4 {
			return nil, fmt.Errorf("incorrect clipping section found of length %d: %s", len(lineContent), string(lineContent))
		}

		currentClipping := Clipping{}

		lines := []struct {
			lineType LineType
			text     []byte
		}{
			{
				lineType: LineType_Source,
				text:     components[0],
			},
			{
				lineType: LineType_Description,
				text:     components[1],
			},
			{
				lineType: LineType_Clipping,
				text:     components[3],
			},
		}

		for _, line := range lines {
			err := k.line(line.lineType, bytes.TrimSpace(line.text), &currentClipping)
			if err != nil {
				return nil, fmt.Errorf("error while parsing a line > %w", err)
			}
		}

		output = append(output, currentClipping)
	}

	return output, nil
}

// line processes the given line or set of lines. When the line type is source, description, or
// separator, this will definitely be a single line. But if the lineType is clipping, then it can be
// multiline because we are using SplitN with N = 4.
func (k *KindleClippings) line(lineType LineType, lineText []byte, clipping *Clipping) error {
	switch lineType {
	case LineType_Source:
		notPrint := func(r rune) bool {
			return !unicode.IsPrint(r)
		}

		// Trim all non-printable characters from the source. This line tends to start with \uFEFF,
		// for some books. This might be because of an older Kindle software version which used that
		// character as a separator, or to indicate the nature of the text that is inside each
		// clipping section.
		clipping.Source = strings.TrimFunc(string(lineText), notPrint)
	case LineType_Description:
		var variationErrors []error
		for variantNum, variation := range KindleDescriptionLineVariations {
			matches := variation.Matcher.FindSubmatchIndex(lineText)
			k.logger.Debug("k.Line > description", zap.Ints("matches", matches), zap.ByteString("description_text", lineText))
			if len(matches) != variation.RequiredMatchCount {
				variationErrors = append(variationErrors, fmt.Errorf(`description line malformed with variant %d: "%s"`, variantNum, lineText))
				continue
			}

			clippingType := string(lineText[matches[variation.Type[0]]:matches[variation.Type[1]]])
			switch clippingType {
			case "Highlight", "ハイライト":
				clipping.Type = ClippingType_Highlight
			case "Note", "メモ":
				clipping.Type = ClippingType_Note
			}

			var err error

			if variation.Page[0] != -1 {
				clipping.Page = string(lineText[matches[variation.Page[0]]:matches[variation.Page[1]]])
			}

			clipping.LocationInSource.Start, err = strconv.Atoi(string(lineText[matches[variation.LocationInSourceStart[0]]:matches[variation.LocationInSourceStart[1]]]))
			if err != nil {
				return fmt.Errorf(`description line > start location could not be parsed from the line: "%s" > %w`, lineText, err)
			}

			if matches[variation.LocationInSourceEnd[0]] != -1 {
				clipping.LocationInSource.End, err = strconv.Atoi(string(lineText[matches[variation.LocationInSourceEnd[0]]:matches[variation.LocationInSourceEnd[1]]]))
				if err != nil {
					return fmt.Errorf(`description line > end location could not be parsed from the line: "%s" > %w`, lineText, err)
				}
			}

			if variation.CreateTime[0] != -1 {
				// creationTime = 2023年5月15日月曜日 20:45:04
				creationTime := lineText[matches[variation.CreateTime[0]]:matches[variation.CreateTime[1]]]
				timeToParse := creationTime

				if variantNum == int(Variant_Japanese) {
					// dateDay = 2023年5月15日月曜日
					dateDay, _, _ := bytes.Cut(creationTime, []byte(" "))

					// We need to remove the last three runes. Not the last three bytes.
					// dateOnly = 2023年5月15日
					dateDayAsRunes := bytes.Runes(dateDay)
					var dateOnly string
					for _, v := range dateDayAsRunes[:len(dateDayAsRunes)-3] {
						dateOnly += string(v)
					}

					timeToParse = []byte(dateOnly)
				}

				clipping.CreateTime, err = time.ParseInLocation(variation.CreateTimeFormat, string(timeToParse), time.Local)
				if err != nil {
					return fmt.Errorf(`description line > creation time could not be parsed from the line: "%s" > %w`, lineText, err)
				}
			}

			break
		}

		if len(variationErrors) == len(KindleDescriptionLineVariations) {
			return fmt.Errorf(`description malformed with all variants > %v`, variationErrors)
		}

	case LineType_Clipping:
		clipping.Text = string(bytes.TrimSpace(
			bytes.TrimSuffix(
				lineText,
				[]byte(KindleClippingsSeparator),
			),
		),
		)
	}

	return nil
}

// scanUsingKindleClippingsSeparator is a function which matches the separator function required by
// bufio.Scanner and uses the KindleClippingSeparator (={10}) as the separator between blocks of
// text.
func (k *KindleClippings) scanUsingKindleClippingsSeparator(data []byte, atEOF bool) (advance int, token []byte, err error) {
	k.logger.Debug("bufio separator function", zap.Bool("at_eof", atEOF), zap.Int("data_length", len(data)), zap.ByteString("data_content", data))

	// Nothing to read, so just stop reading.
	if atEOF && len(data) == 0 {
		k.logger.Debug("bufio separator function > at EOF and no more data")
		return 0, nil, nil
	}

	// We are not at EOF or there is *some* data. This data might be in the `data` byteString.

	// Look for the separator in the already read bytestring.
	if loc := KindleClippingsSeparatorMatcher.FindIndex(data); loc != nil {
		// Separator found. We have at least 1 complete clipping section here.
		k.logger.Debug("bufio separator function > found separator", zap.Ints("locations_of_separator", loc))
		return loc[1], data[:loc[0]], nil
	}

	// If we're at EOF but we did not find the separator, then this might be the last (possibly
	// malformed) clipping section. So, return it as is and advance until the end of the underlying
	// data.
	if atEOF {
		k.logger.Debug("bufio separator function > at eof with final block of data", zap.Int("data_lenght", len(data)), zap.ByteString("data_content", data))
		return len(data), data, nil
	}

	// We are not at EOF and we don't see a separator either. So, ask the scanner to read more data.
	k.logger.Debug("bufio separator function > not at eof; not enough data", zap.Int("data_lenght", len(data)), zap.ByteString("data_content", data))
	return 0, nil, nil
}

// isException looks at a clipping section, which has been split using newline already and
// identifies whether the clipping should be treated as an exception and skipped over.
func (k *KindleClippings) isException(comps [][]byte) bool {
	// Bookmark type clippings are included in Kindle's My Clippings text file and have only 2
	// lines. The first line contains the source, whereas the second line contains the Bookmark,
	// which has location information. We should ignore these.
	if len(comps) == 2 &&
		(bytes.HasPrefix(comps[1], []byte("- Your Bookmark")) ||
			strings.Contains(string(comps[1]), `ブックマーク`)) {
		return true
	}

	// Kindle has an annoying feature which prevents the text of clippings that exceded the 10% of a
	// book has been clipped limitation from showing up in the My Clippings.txt file. So, we will
	// ignore them from the parsed YAML for the time being.
	//
	// Use bookcision.js to get these clippings and merge the two files together somehow.
	if k.RemoveClippingLimitClippings && len(comps) == 4 &&
		(bytes.Contains(comps[3], []byte(KindleClippingLimitMessage))) {
		return true
	}

	return false
}
