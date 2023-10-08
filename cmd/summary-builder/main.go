package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"github.com/icyflame/kindle-my-clippings-parser/internal/summarizer"
	"github.com/icyflame/kindle-my-clippings-parser/internal/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	ExitOK = iota
	ExitErr
)

func main() {
	err := _main()
	if err != nil {
		fmt.Printf("error > %v\n", err)
		os.Exit(ExitErr)
	}
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

	logger.Info("read clippings from YAML file", zap.String("file", inputFilePath))

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

	var sourceName string
	sources := make(map[string]bool)
	for _, clipping := range clippings {
		if sourceFilterRx == nil || sourceFilterRx.MatchString(clipping.Source) {
			if sourceName == "" {
				sourceName = clipping.Source
			}
			sources[clipping.Source] = true
			if len(sources) > 1 {
				return fmt.Errorf("invalid source filter: Summary builder can build the summary for only one source at a time")
			}
		}
	}

	logger.Info("build summary for source", zap.String("source", sourceName))

	summaryCreator := summarizer.KindleCreator{
		Logger: logger.With(zap.String("component", "summarizer")),
	}

	summary, err := summaryCreator.Summarize(utils.FilterBySource(clippings, sourceName))
	if err != nil {
		return fmt.Errorf("could not create summary of source '%s' > %w", sourceName, err)
	}

	tmpl, err := template.ParseFiles("./cmd/summary-builder/summary.org.tmpl", "./cmd/summary-builder/chapter.org.tmpl")
	if err != nil {
		return fmt.Errorf("error reading the template files > %w", err)
	}

	err = tmpl.Execute(os.Stdout, summary)
	if err != nil {
		return fmt.Errorf("error while executing the templates > %w", err)
	}

	return nil
}
