package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func readSessionJSON(source string, stdin io.Reader) ([]byte, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("requires non-empty JSON")
	}

	var data []byte
	var err error
	switch {
	case source == "-":
		data, err = readLimited(stdin, maxSessionJSONBytes)
	case strings.HasPrefix(source, "@"):
		path := strings.TrimSpace(strings.TrimPrefix(source, "@"))
		if path == "" {
			return nil, fmt.Errorf("@file path is empty")
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, openErr)
		}
		defer file.Close()
		data, err = readLimited(file, maxSessionJSONBytes)
	default:
		if len([]byte(source)) > maxSessionJSONBytes {
			return nil, fmt.Errorf("JSON exceeds %d bytes", maxSessionJSONBytes)
		}
		data = []byte(source)
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}
