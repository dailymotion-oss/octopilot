package yqlib

import (
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v3"
)

type InputFormat uint

const (
	YamlInputFormat = 1 << iota
	XMLInputFormat
	PropertiesInputFormat
	Base64InputFormat
	JsonInputFormat
	CSVObjectInputFormat
	TSVObjectInputFormat
)

type Decoder interface {
	Init(reader io.Reader)
	Decode(node *yaml.Node) error
}

func InputFormatFromString(format string) (InputFormat, error) {
	switch format {
	case "yaml", "y":
		return YamlInputFormat, nil
	case "xml", "x":
		return XMLInputFormat, nil
	case "props", "p":
		return PropertiesInputFormat, nil
	case "json", "ndjson", "j":
		return JsonInputFormat, nil
	case "csv", "c":
		return CSVObjectInputFormat, nil
	case "tsv", "t":
		return TSVObjectInputFormat, nil
	default:
		return 0, fmt.Errorf("unknown format '%v' please use [yaml|xml|props]", format)
	}
}
