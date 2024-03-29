package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"text/template"

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
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Input file should be the YAML file that is output by the cmd/parse command in this project.")
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

	type TemplateData struct {
		ClippingPairs []parser.Clippings
	}

	var data TemplateData
	data.ClippingPairs = make([]parser.Clippings, 0)

	for clippingKey, clippings := range clippingIndex {
		count := len(clippings)

		if count == 2 {
			logger.Info("duplicates identified", zap.String("key", clippingKey), zap.Int("count", count))
			var sortedClippings parser.Clippings = clippings
			sort.Sort(parser.Clippings(sortedClippings))
			if sortedClippings[0].LocationInSource.Start == 9948 || sortedClippings[0].LocationInSource.Start == 16734 || sortedClippings[0].LocationInSource.Start == 5662 {
				data.ClippingPairs = append(data.ClippingPairs, sortedClippings)
			}
		}
	}

	tmpl, err := template.ParseFiles("./cmd/identify-duplicate-pairs/identify-duplicate-pairs.html.tmpl")
	if err != nil {
		return fmt.Errorf("error while parsing the input template file > %w", err)
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		return fmt.Errorf("error while executing text template > %w", err)
	}

	return nil
}
