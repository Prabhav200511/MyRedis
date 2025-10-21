package internal

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

type AOF struct {
	filename string
	mu       sync.Mutex
	file     *os.File
	writer   *bufio.Writer
}

func NewAOF(filename string) (*AOF, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &AOF{
		filename: filename,
		file:     file,
		writer:   bufio.NewWriter(file),
	}, nil
}

// AppendCommand writes a single command line to the AOF
func (a *AOF) AppendCommand(cmd string, args ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	line := cmd
	for _, arg := range args {
		line += " " + arg
	}
	line += "\n"

	if _, err := a.writer.WriteString(line); err != nil {
		return err
	}
	return a.writer.Flush()
}

// Replay reads the AOF and applies commands to the store
func (a *AOF) Replay(store *Store) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	_, err := a.file.Seek(0, 0)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(a.file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		cmd := strings.ToUpper(parts[0])
		switch cmd {
		case "SET":
			if len(parts) >= 3 {
				store.Set(parts[1], parts[2])
			}
		case "DEL":
			if len(parts) >= 2 {
				store.mu.Lock()
				delete(store.data, parts[1])
				store.mu.Unlock()
			}
		}
	}
	return scanner.Err()
}

// Close the AOF file
func (a *AOF) Close() {
	a.writer.Flush()
	a.file.Close()
}
