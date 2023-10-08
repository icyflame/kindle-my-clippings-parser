package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/icyflame/kindle-my-clippings-parser/internal/duplicates"
	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"go.uber.org/zap"

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
	var inputFilePath, outputFilePath string
	var verbose, removeDuplicates, removeClippingLimit bool
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Preferably the My Clippings.txt file from Kindle")
	flag.StringVar(&outputFilePath, "output-file-path", "", "Output file. Output will be written in the YAML format.")
	flag.BoolVar(&removeClippingLimit, "remove-clipping-limit", false, "Remove clippings which indicate that the clipping text was not saved to the text file")
	flag.BoolVar(&removeDuplicates, "remove-duplicates", false, "Remove duplicate clippings of type Highlight from the generated YAML file")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if inputFilePath == "" {
		return errors.New("input file path must be non-empty")
	}

	if _, err := os.Stat(inputFilePath); err != nil {
		return fmt.Errorf("input file must point to a valid file > %w", err)
	}

	if outputFilePath == "" {
		return errors.New("output file path must be non-empty")
	}

	if _, err := os.Stat(outputFilePath); err == nil {
		return errors.New("output file path must not exist before this script runs")
	}

	logger, err := zap.NewProduction()
	if verbose {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return fmt.Errorf("could not create logger > %w", err)
	}

	processor := parser.NewParserWithLogger(inputFilePath, removeClippingLimit, logger.With(zap.String("component", "processor")))

	clippings, err := processor.Parse()
	if err != nil {
		return fmt.Errorf("error while parsing clippings file > %w", err)
	}

	logger.Info("Read clippings from file", zap.Int("clipping_count", len(clippings)))

	if removeDuplicates {
		deduper := duplicates.RetainLatest{
			Logger: logger.With(zap.String("component", "deduper")),
		}
		dedupedClippings, err := deduper.Delete(clippings)
		if err != nil {
			return fmt.Errorf("error while removing duplicates from the clippings set > %w", err)
		}
		clippings = dedupedClippings
		logger.Info("Deduplicate clippings", zap.Int("clipping_count", len(clippings)))
	}

	sort.Sort(clippings)

	outputFile, err := os.Create(outputFilePath)
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
