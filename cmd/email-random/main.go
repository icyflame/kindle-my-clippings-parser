package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"

	"github.com/icyflame/kindle-my-clippings-parser/internal/parser"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	ExitOK = iota
	ExitErr
)

const WantClippingType = parser.ClippingType_Highlight

func main() {
	err := _main()
	if err != nil {
		fmt.Printf("error > %v\n", err)
		os.Exit(ExitErr)
	}
}

func _main() error {
	var inputFilePath string
	var verbose bool
	flag.StringVar(&inputFilePath, "input-file-path", "", "Input file. Preferably the My Clippings.txt file from Kindle")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if inputFilePath == "" {
		return errors.New("input file path must be non-empty")
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

	count := 0
	for _, c := range clippings {
		if c.Type == WantClippingType {
			count++
		}
	}

	logger.Info("clippings of wanted type", zap.Any("want_type", WantClippingType), zap.Int("clipping_count", count))

	selectedClipping, err := chooseHighlight(clippings)
	if err != nil {
		return fmt.Errorf("could not select a random highlight > %w", err)
	}

	logger.Debug("selected clipping", zap.Any("selected", selectedClipping))

	return nil
}

func chooseHighlight(clippings parser.Clippings) (parser.Clipping, error) {
	for i := 0; i < 10; i++ {
		randInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(clippings))))
		if err != nil {
			return parser.Clipping{}, fmt.Errorf("could not generate a random number between 0 and %d > %w", len(clippings), err)
		}

		temporary := clippings[int(randInt.Int64())]
		if temporary.Type == WantClippingType {
			return temporary, nil
		}
	}

	return parser.Clipping{}, fmt.Errorf("could not get a highlight despite 10 attempts")
}
