package internal

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ParseRESP decodes one RESP command from the connection (like ["SET", "key", "value"])
func ParseRESP(reader *bufio.Reader) ([]string, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if prefix != '*' {
		return nil, fmt.Errorf("expected array ('*'), got %q", prefix)
	}

	lengthLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(lengthLine))
	if err != nil {
		return nil, err
	}

	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if b != '$' {
			return nil, fmt.Errorf("expected bulk string ('$'), got %q", b)
		}

		lenLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		strLen, err := strconv.Atoi(strings.TrimSpace(lenLine))
		if err != nil {
			return nil, err
		}

		buf := make([]byte, strLen+2)
		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return nil, err
		}
		parts = append(parts, string(buf[:strLen]))
	}

	return parts, nil
}

// Encoding helpers for RESP replies
func EncodeSimpleString(s string) string { return fmt.Sprintf("+%s\r\n", s) }
func EncodeBulkString(s string) string   { return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s) }
func EncodeInteger(n int) string         { return fmt.Sprintf(":%d\r\n", n) }
func EncodeError(msg string) string      { return fmt.Sprintf("-%s\r\n", msg) }
