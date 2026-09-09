package mobile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func LoadHistory(path string, maxEntries int) ([]string, error) {
	if path == "" {
		return []string{}, nil
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open REPL history: %w", err)
	}
	defer file.Close()
	result := []string{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1_000_000)
	for scanner.Scan() {
		var entry string
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("decode REPL history: %w", err)
		}
		result = append(result, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if maxEntries > 0 && len(result) > maxEntries {
		result = result[len(result)-maxEntries:]
	}
	return result, nil
}

func AppendHistory(path, entry string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create REPL history directory: %w", err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open REPL history: %w", err)
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func ClearHistory(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear REPL history: %w", err)
	}
	return nil
}
