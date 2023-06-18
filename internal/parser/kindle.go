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
)

type KindleClippings struct {
	FilePath string
}

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

// Alias Grace (Atwood, Margaret)
// - Your Highlight on page 22 | location 281-283 | Added on Sunday, 5 May 2019 10:23:20
// - Your Highlight on page ix | location 341-344 | Added on Saturday, 25 January 2020 10:47:54

// They were bell-shaped and ruffled, gracefully waving and lovely under the sea; but if they washed up on the beach and dried out in the sun there was nothing left of them. And that is what the ladies are like: mostly water.
// ^^ this can be multiline
// ==========
func (k *KindleClippings) Parse() (Clippings, error) {
	clippings, err := os.Open(k.FilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open the clippings file > %w", err)
	}

	defer clippings.Close()

	var output []Clipping

	scanner := bufio.NewScanner(clippings)
	scanner.Split(ScanUsingUTF8FEFFAsDelimiter)

	// Initial character of Kindle's file is also a separator. So, we have to scan at least once to
	// get rid of that section.
	scanner.Scan()

	for scanner.Scan() {
		// lineText := scanner.Text()
		// components := strings.SplitN(lineText, "\n", 4)

		lineContent := scanner.Bytes()
		lineContent = bytes.TrimSpace(lineContent)
		components := bytes.SplitN(lineContent, []byte{'\n'}, 4)

		if k.IsException(components) {
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
				// Each source line starts with 2 bytes which are there to denote the encoding of that
				// particular clippings file.
				text: components[0],
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
			err := k.Line(line.lineType, bytes.TrimSpace(line.text), &currentClipping)
			if err != nil {
				return nil, fmt.Errorf("error while parsing a line > %w", err)
			}
		}

		output = append(output, currentClipping)
	}

	return output, nil
}

// Line ...
func (k *KindleClippings) Line(lineType LineType, lineText []byte, clipping *Clipping) error {
	switch lineType {
	case LineType_Source:
		notPrint := func(r rune) bool {
			return !unicode.IsPrint(r)
		}

		clipping.Source = strings.TrimFunc(string(lineText), notPrint)
	case LineType_Description:
		var variationErrors []error
		for variantNum, variation := range KindleDescriptionLineVariations {
			matches := variation.Matcher.FindSubmatchIndex(lineText)
			// fmt.Printf("description: %#v\n", matches)
			// fmt.Printf("description: %s\n", string(lineText))
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

const KindleClippingsSeparator = "=========="

var KindleClippingsSeparatorMatcher *regexp.Regexp = regexp.MustCompile(`={10}`)

type LineType int

const (
	LineType_Invalid LineType = iota
	LineType_Source
	LineType_Description
	LineType_Empty
	LineType_Clipping
)

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

type Variant int

const (
	Variant_English_BooksWithPageNum Variant = iota
	Variant_English_BooksWithoutPageNum
	Variant_Japanese
)

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
}

// ScanUsingUTF8FEFFAsDelimiter is a special character that is used in Kindle's clippings files.
func ScanUsingUTF8FEFFAsDelimiter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// fmt.Printf("given data of length %d (at EOF: %v): %s\n", len(data), atEOF, string(data))
	// Nothing to read, so just end this.
	if atEOF && len(data) == 0 {
		// fmt.Println("---")
		// fmt.Printf("Scan: at eof and no data")
		return 0, nil, nil
	}

	// Look for the separator
	if loc := KindleClippingsSeparatorMatcher.FindIndex(data); loc != nil {
		// fmt.Println("---")
		// fmt.Println("Scan: ", len(data), loc[1]+2, len(data[:loc[0]]))
		return loc[1], data[:loc[0]], nil
	}

	// If we're at EOF and we still did not find the separator, then we don't have anything to
	// return
	if atEOF {
		// fmt.Println("---")
		// fmt.Println("Scan: at eof and returning remaining data: ", len(data))
		return len(data), data, nil
	}

	// We are not at EOF and we don't see a separator either
	return 0, nil, nil
}

// IsException ...
func (k *KindleClippings) IsException(comps [][]byte) bool {
	if len(comps) == 2 &&
		(bytes.HasPrefix(comps[1], []byte("- Your Bookmark")) ||
			strings.Contains(string(comps[1]), `ブックマーク`)) {
		return true
	}

	return false
}
