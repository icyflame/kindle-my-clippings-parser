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
	var verbose bool
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Input file should be the YAML file that is output by the cmd/parse command in this project.")
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

	logger.Info("Reading clippings from YAML file", zap.String("file", inputFilePath))

	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return fmt.Errorf("could not create output yaml file > %w", err)
	}
	defer inputFile.Close()

	reader := yaml.NewDecoder(inputFile)
	var clippings parser.Clippings
	if err := reader.Decode(&clippings); err != nil {
		return fmt.Errorf("could not encode parsed clippings into YAML > %w", err)
	}

	logger.Info("read clippings from parsed YAML file", zap.Int("clipping_count", len(clippings)))

	deduper := duplicates.RetainLatest{
		Logger: logger.With(zap.String("component", "deduper")),
	}

	dedupedClippings, err := deduper.Delete(clippings)
	if err != nil {
		return fmt.Errorf("error while removing duplicates from the clippings set > %w", err)
	}

	logger.Info("deduplicate clippings", zap.Int("clipping_count", len(dedupedClippings)))

	sort.Sort(dedupedClippings)

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("could not create output yaml file > %w", err)
	}
	defer outputFile.Close()

	writer := yaml.NewEncoder(outputFile)
	defer writer.Close()
	if err := writer.Encode(dedupedClippings); err != nil {
		return fmt.Errorf("could not encode parsed clippings into YAML > %w", err)
	}

	return nil
}
