package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
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
	lineType := LineType_Source
	currentClipping := Clipping{}

	for scanner.Scan() {
		lineText := scanner.Text()

		if lineText == KindleClippingsSeparator {
			output = append(output, currentClipping)
			currentClipping = Clipping{}
			lineType = LineType_Source
			continue
		}

		switch lineType {
		case LineType_Source:
			fmt.Printf("found source line: %s\n", lineText)
			lineType = LineType_Description
		case LineType_Description:
			fmt.Printf("found description line: %s\n", lineText)
			lineType = LineType_Empty
		case LineType_Empty:
			lineType = LineType_Clipping
		case LineType_Clipping:
			fmt.Printf("found clipping line: %s\n", lineText)
			// do not change lineType here
		}
	}

	fmt.Printf("reached end")

	return output, nil
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
