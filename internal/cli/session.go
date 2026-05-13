package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func readSessionJSON(source string, stdin io.Reader, maxBytes int) ([]byte, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("requires non-empty JSON")
	}

	var data []byte
	var err error
	switch {
	case source == "-":
		data, err = readLimited(stdin, int64(maxBytes))
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
		data, err = readLimited(file, int64(maxBytes))
	default:
		if len([]byte(source)) > maxBytes {
			return nil, fmt.Errorf("JSON exceeds %d bytes", maxBytes)
		}
		data = []byte(source)
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	if reader == nil {
		return nil, fmt.Errorf("input is unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("JSON exceeds %d bytes", maxBytes)
	}
	return data, nil
}
