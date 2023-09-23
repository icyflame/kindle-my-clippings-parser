package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

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
	var inputFilePath string
	var outputFilePath string
	var verbose bool
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Preferably the My Clippings.txt file from Kindle")
	flag.StringVar(&outputFilePath, "output-file-path", "", "Output file. Output will be written in the YAML format.")
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

	processor := parser.NewParserWithLogger(inputFilePath, logger.With(zap.String("component", "processor")))

	clippings, err := processor.Parse()
	if err != nil {
		return fmt.Errorf("error while parsing clippings file > %w", err)
	}

	logger.Info("Read clippings from file", zap.Int("clipping_count", len(clippings)))

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
