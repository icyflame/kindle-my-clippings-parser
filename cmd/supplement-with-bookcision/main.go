package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"github.com/icyflame/kindle-my-clippings-parser/internal/supplementer"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	ExitOK int = iota
	ExitErr
)

func main() {
	err := _main()
	if err != nil {
		log.Println(fmt.Errorf("error from _main: %w", err))
		os.Exit(ExitErr)
	}

	os.Exit(ExitOK)
}

func _main() error {
	var inputFilePath, outputFilePath, supplementFilePath, sourceFilter string
	var verbose bool
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Preferably the My Clippings.txt file from Kindle")
	flag.StringVar(&outputFilePath, "output-file-path", "", "Output file. Output will be written in the YAML format.")
	flag.StringVar(&supplementFilePath, "supplement-file-path", "", "JSON file with all the clippings, exported using Bookcision")
	flag.StringVar(&sourceFilter, "source-filter", "", "Regular expression for filtering the source of clippings")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if inputFilePath == "" {
		flag.PrintDefaults()
		return errors.New("input file path must be non-empty")
	}

	if supplementFilePath == "" {
		flag.PrintDefaults()
		return errors.New("supplement file path must be non-empty")
	}

	var sourceFilterRx *regexp.Regexp
	if sourceFilter != "" {
		sfRx, err := regexp.Compile(sourceFilter)
		if err != nil {
			return fmt.Errorf("supplied source filter '%s' is invalid > %w", sourceFilter, err)
		}
		sourceFilterRx = sfRx
	}

	if _, err := os.Stat(inputFilePath); err != nil {
		return fmt.Errorf("input file must point to a valid file > %w", err)
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

	bookcisionParser := parser.BookcisionClippings{
		FilePath: supplementFilePath,
		Logger:   logger.With(zap.String("component", "parser"), zap.String("subcomponent", "bookcision")),
	}

	bookcisionClippings, err := bookcisionParser.Parse()
	if err != nil {
		return fmt.Errorf("could not parse bookcision JSON file '%s' > %w", supplementFilePath, err)
	}

	supplementer := supplementer.Bookcision{
		SourceRegex: sourceFilterRx.Copy(),
		Logger:      logger.With(zap.String("component", "supplementer")),
	}

	supplementedClippings, err := supplementer.Merge(clippings, bookcisionClippings)
	if err != nil {
		return fmt.Errorf("could not supplement kindle with bookcision > %w", err)
	}

	sort.Sort(supplementedClippings)

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("could not create output yaml file > %w", err)
	}
	defer outputFile.Close()

	writer := yaml.NewEncoder(outputFile)
	defer writer.Close()
	if err := writer.Encode(supplementedClippings); err != nil {
		return fmt.Errorf("could not encode parsed supplementedClippings into YAML > %w", err)
	}

	return nil
}
