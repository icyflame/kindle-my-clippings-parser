package parser

type Parser interface {
	Parse() (Clippings, error)
}

func NewParser(inputFilePath string) Parser {
	return &KindleClippings{
		FilePath: inputFilePath,
	}

}
