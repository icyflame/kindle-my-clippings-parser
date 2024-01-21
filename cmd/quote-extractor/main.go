package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"github.com/icyflame/kindle-my-clippings-parser/internal/utils"
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

	if sourceFilter == "" {
		flag.PrintDefaults()
		return errors.New("source filter can not be empty")
	}

	var sourceFilterRx *regexp.Regexp
	sfRx, err := regexp.Compile(sourceFilter)
	if err != nil {
		return fmt.Errorf("supplied source filter '%s' is an invalid regular expression > %w", sourceFilter, err)
	}
	sourceFilterRx = sfRx

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

	logger.Info("reading clippings from YAML file", zap.String("file", inputFilePath))

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

	clippings = utils.FilterBySourceRegex(clippings, sourceFilterRx.Copy())

	sort.Sort(clippings)

	quoteClippings := make(parser.Clippings, 0)
	for i, clipping := range clippings {
		if clipping.Type == parser.ClippingType_Note && strings.HasPrefix(clipping.Text, "#quote") {
			if clippings[i-1].Type == parser.ClippingType_Highlight {
				quoteClippings = append(quoteClippings, clippings[i-1])
			}
		}
	}

	type TemplateData struct {
		Clippings parser.Clippings
	}

	var data TemplateData
	data.Clippings = quoteClippings

	tmpl, err := template.ParseFiles("./cmd/quote-extractor/quotes.org.tmpl")
	if err != nil {
		return fmt.Errorf("error while parsing the input template file > %w", err)
	}

	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		return fmt.Errorf("error while executing text template > %w", err)
	}

	return nil
}
