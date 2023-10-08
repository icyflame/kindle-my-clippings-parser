package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
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
	var inputFilePath, sourceFilter string
	var verbose bool
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Preferably the My Clippings.txt file from Kindle")
	flag.StringVar(&sourceFilter, "source-filter", "", "Regular expression for filtering the source of clippings")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if inputFilePath == "" {
		flag.PrintDefaults()
		return errors.New("input file path must be non-empty")
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

	logger.Info("read clippings", zap.Int("clipping_count", len(clippings)))

	clippingIndex := make(map[string][]parser.Clipping)
	for _, clipping := range clippings {
		if sourceFilterRx != nil && !sourceFilterRx.MatchString(clipping.Source) {
			continue
		}

		if clipping.Type != parser.ClippingType_Note {
			continue
		}

		clippingKey := fmt.Sprintf("%s-%d-%d", clipping.Source, clipping.Type, clipping.LocationInSource.Start)

		if _, ok := clippingIndex[clippingKey]; !ok {
			clippingIndex[clippingKey] = make([]parser.Clipping, 0)
		}

		clippingIndex[clippingKey] = append(clippingIndex[clippingKey], clipping)
	}

	for key, clippings := range clippingIndex {
		count := len(clippings)
		if count > 1 {
			fmt.Printf("%s = %d\n", key, count)
		}
	}

	return nil
}
