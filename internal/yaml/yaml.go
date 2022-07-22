// Package yaml provides functions to work with YAML content
package yaml

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strings"
)

// ExtractLeadingContentForYQ extracts the leading content (comment, frontmatter, ...) of a YAML file, for the YQ lib
// It is copied from the `processReadStream` func in the yq lib (because it's not public)
// It is a workaround for go-yaml to persist header content
func ExtractLeadingContentForYQ(input io.Reader) (io.Reader, string, error) {
	var (
		commentLineRegEx = regexp.MustCompile(`^\s*#`)
		reader           = bufio.NewReader(input)
		sb               strings.Builder
	)
	for {
		peekBytes, err := reader.Peek(3)
		if errors.Is(err, io.EOF) {
			// EOF are handled else where..
			return reader, sb.String(), nil
		} else if err != nil {
			return reader, sb.String(), err
		} else if string(peekBytes) == "---" {
			_, err := reader.ReadString('\n')
			sb.WriteString("$yqDocSeperator$\n")
			if errors.Is(err, io.EOF) {
				return reader, sb.String(), nil
			} else if err != nil {
				return reader, sb.String(), err
			}
		} else if commentLineRegEx.MatchString(string(peekBytes)) {
			line, err := reader.ReadString('\n')
			sb.WriteString(line)
			if errors.Is(err, io.EOF) {
				return reader, sb.String(), nil
			} else if err != nil {
				return reader, sb.String(), err
			}
		} else {
			return reader, sb.String(), nil
		}
	}
}
