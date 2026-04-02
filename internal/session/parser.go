package session

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

// Parser parses a provider-specific JSONL file into canonical sessions and messages.
type Parser interface {
	// ParseFile reads a JSONL file and returns a session with messages.
	ParseFile(path string) (*Session, []Message, error)
	// Provider returns the provider name this parser handles.
	Provider() string
}

// ParseJSONL reads a JSONL file line by line and calls the handler for each line.
func ParseJSONL(path string, handler func(line []byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	return ParseJSONLReader(f, handler)
}

// ParseJSONLReader reads JSONL from an io.Reader.
func ParseJSONLReader(r io.Reader, handler func(line []byte) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := handler(line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
