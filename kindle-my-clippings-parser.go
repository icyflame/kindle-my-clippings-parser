package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"
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

	return nil
}

type Location struct {
	Start int
	End   int
}

type ClippingType int

const (
	ClippingType_None ClippingType = iota
	ClippingType_Highlight
	ClippingType_Note
)

type Clipping struct {
	Source           string
	Author           string
	Type             ClippingType
	Page             int
	LocationInSource Location
	CreateTime       time.Time
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

		return nil
	}

	return nil
}

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
	// Each source line is the starting of a "pseudo-file" and has
	KindleSource regexp.Regexp = *regexp.MustCompile(`^(.+) \((.+)\)$`)
)

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
