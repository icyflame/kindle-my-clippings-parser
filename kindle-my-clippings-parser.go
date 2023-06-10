package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ExitOK = iota
	ExitErr
)

// main ...
func main() {
	err := _main()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(ExitErr)
	}
	os.Exit(ExitOK)
}

func _main() error {
	var inputFilePath string
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Preferably the My Clippings.txt file from Kindle")
	flag.Parse()

	if inputFilePath == "" {
		return errors.New("input file path must be non-empty")
	}

	if _, err := os.Stat(inputFilePath); err != nil {
		return fmt.Errorf("input file must point to a valid file > %w", err)
	}

	parser := KindleClippings{
		FilePath: inputFilePath,
	}

	clippings, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("error while parsing clippings file > %w", err)
	}
	fmt.Printf("Clippings output:\n\n%#v", clippings)

	outputFile, err := os.Create("parsed-clippings.yaml")
	if err != nil {
		return fmt.Errorf("could not create output yaml file > %w", err)
	}
	defer outputFile.Close()

	writer := yaml.NewEncoder(outputFile)
	defer writer.Close()
	if err := writer.Encode(clippings); err != nil {
		return fmt.Errorf("could not encode parsed clippings into YAML > %w", err)
	}

	return nil
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
	Source           string       `yaml:"source"`
	Author           string       `yaml:"author"`
	Type             ClippingType `yaml:"type"`
	Page             int          `yaml:"page"`
	LocationInSource Location     `yaml:"location_in_source"`
	CreateTime       time.Time    `yaml:"create_time"`
	Text             string       `yaml:"text"`
}

type Clippings []Clipping

type Parser interface {
	Parse() (Clippings, error)
}

type KindleClippings struct {
	FilePath string
}

// Alias Grace (Atwood, Margaret)
// - Your Highlight on page 22 | location 281-283 | Added on Sunday, 5 May 2019 10:23:20

// They were bell-shaped and ruffled, gracefully waving and lovely under the sea; but if they washed up on the beach and dried out in the sun there was nothing left of them. And that is what the ladies are like: mostly water.
// ^^ this can be multiline
// KindleClippingsSeparator
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
		components := bytes.SplitN(lineContent, []byte{'\n'}, 4)

		if len(components) != 4 {
			return nil, fmt.Errorf("incorrect clipping section found of length %d", len(lineContent))
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
				text: components[0][2:],
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
		matches := KindleSource.FindSubmatchIndex(lineText)
		if len(matches) != 6 {
			return fmt.Errorf(`source line malformed: "%s"`, lineText)
		}

		clipping.Source = string(lineText[matches[2]:matches[3]])
		clipping.Author = string(lineText[matches[4]:matches[5]])
	case LineType_Description:
		var variationErrors []error
		for variantNum, variation := range KindleDescriptionLineVariations {
			matches := variation.Matcher.FindSubmatchIndex(lineText)
			if len(matches) != variation.RequiredMatchCount {
				variationErrors = append(variationErrors, fmt.Errorf(`description line malformed with variant %d: "%s"`, variantNum, lineText))
				continue
			}

			clippingType := string(lineText[matches[variation.Type[0]]:matches[variation.Type[1]]])
			switch clippingType {
			case "Highlight":
				clipping.Type = ClippingType_Highlight
			case "Note":
				clipping.Type = ClippingType_Note
			}

			fmt.Printf("%#v\n", matches)
			fmt.Printf("%s\n", lineText)

			var err error

			if variation.Page[0] != -1 {
				clipping.Page, err = strconv.Atoi(string(lineText[matches[variation.Page[0]]:matches[variation.Page[1]]]))
				if err != nil {
					return fmt.Errorf(`description line > page number could not be parsed from the line: "%s" > %w`, lineText, err)
				}
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

			creationTime := lineText[matches[variation.CreateTime[0]]:matches[variation.CreateTime[1]]]
			clipping.CreateTime, err = time.ParseInLocation(variation.CreateTimeFormat, string(creationTime), time.Local)
			if err != nil {
				return fmt.Errorf(`description line > creation time could not be parsed from the line: "%s" > %w`, lineText, err)
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

type LineType int

const (
	LineType_Invalid LineType = iota
	LineType_Source
	LineType_Description
	LineType_Empty
	LineType_Clipping
)

var (
	// KindleSource is a regular expression representing the first line of every Kindle highlight/note.
	//
	// Sample line:
	// "Alias Grace (Atwood, Margaret)"
	KindleSource regexp.Regexp = *regexp.MustCompile(`^(.+) \((.+)\)$`)
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

var KindleDescriptionLineVariations []KindleDescriptionLineVariation = []KindleDescriptionLineVariation{
	// Sample line: Variation 1: Books with page numbers
	// "- Your Highlight on page 373 | location 5709-5720 | Added on Sunday, 16 April 2023 10:13:54"
	// "- Your Note on page 286 | location 4371 | Added on Saturday, 15 April 2023 12:51:43"
	{
		Matcher:               regexp.MustCompile(`^- Your (.+?) on page (\d+) \| location (\d+)-?(\d+)? \| Added on (.+)$`),
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
	{
		Matcher:               regexp.MustCompile(`- (\d+)ページ|位置No. (\d+)-?(\d+)?の(.+) |作成日: (.+)`),
		RequiredMatchCount:    12,
		Type:                  [2]int{8, 9},
		Page:                  [2]int{2, 3},
		LocationInSourceStart: [2]int{4, 5},
		LocationInSourceEnd:   [2]int{6, 7},
		CreateTime:            [2]int{10, 11},
		CreateTimeFormat:      "2006年1月02日月曜日 15:04:05",
	},
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

// ScanUsingUTF8FEFFAsDelimiter is a special character that is used in Kindle's clippings files.
func ScanUsingUTF8FEFFAsDelimiter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexRune(data, 0xfeff); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}
