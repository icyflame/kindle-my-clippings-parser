package parser

import "go.uber.org/zap"

type Parser interface {
	Parse() (Clippings, error)
}

func NewParser(inputFilePath string) Parser {
	return &KindleClippings{
		FilePath: inputFilePath,
		logger:   zap.NewNop(),
	}
}

func NewParserWithLogger(inputFilePath string, removeClippingLimitMessages bool, logger *zap.Logger) Parser {
	return &KindleClippings{
		FilePath:                     inputFilePath,
		RemoveClippingLimitClippings: removeClippingLimitMessages,
		logger:                       logger,
	}
}
