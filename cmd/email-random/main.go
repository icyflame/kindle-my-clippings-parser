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

	randInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(clippings))))
	if err != nil {
		return fmt.Errorf("could not generate a random number between 0 and %d > %w", len(clippings), err)
	}

	selectedClipping := clippings[int(randInt.Int64())]
	logger.Debug("selected clipping", zap.Any("selected", selectedClipping))

	return nil
}
