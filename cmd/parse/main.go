package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"

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

	processor := parser.NewParser(inputFilePath)

	clippings, err := processor.Parse()
	if err != nil {
		return fmt.Errorf("error while parsing clippings file > %w", err)
	}

	fmt.Printf("Read %d clippings from file", len(clippings))

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
