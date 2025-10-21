package internal

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type Server struct {
	addr  string
	store *Store
}

// NewServer accepts an address and a Store
func NewServer(addr string, store *Store) *Server {
	return &Server{
		addr:  addr,
		store: store,
	}
}

// Start launches the TCP server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	fmt.Println("🧠 MiniRedis listening on", s.addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("connection error:", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// handleConnection reads RESP commands and dispatches them
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		cmdParts, err := ParseRESP(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprint(conn, EncodeError("ERR invalid request"))
			continue
		}

		if len(cmdParts) == 0 {
			continue
		}

		cmd := strings.ToUpper(cmdParts[0])

		switch cmd {
		case "PING":
			fmt.Fprint(conn, EncodeSimpleString("PONG"))

		case "SET":
			if len(cmdParts) < 3 {
				fmt.Fprint(conn, EncodeError("ERR wrong number of arguments"))
				continue
			}
			s.store.Set(cmdParts[1], cmdParts[2])
			fmt.Fprint(conn, EncodeSimpleString("OK"))

		case "GET":
			if len(cmdParts) < 2 {
				fmt.Fprint(conn, EncodeError("ERR wrong number of arguments"))
				continue
			}
			if val, ok := s.store.Get(cmdParts[1]); ok {
				fmt.Fprint(conn, EncodeBulkString(val))
			} else {
				fmt.Fprint(conn, "$-1\r\n")
			}

		case "DEL":
			if len(cmdParts) < 2 {
				fmt.Fprint(conn, EncodeError("ERR wrong number of arguments"))
				continue
			}
			s.store.mu.Lock()
			delete(s.store.data, cmdParts[1])
			s.store.mu.Unlock()
			fmt.Fprint(conn, EncodeInteger(1))

		case "EXPIRE":
			if len(cmdParts) < 3 {
				fmt.Fprint(conn, EncodeError("ERR wrong number of arguments"))
				continue
			}
			key := cmdParts[1]
			seconds, err := strconv.ParseInt(cmdParts[2], 10, 64)
			if err != nil {
				fmt.Fprint(conn, EncodeError("ERR invalid expire time"))
				continue
			}
			ok := s.store.Expire(key, seconds)
			if ok {
				fmt.Fprint(conn, EncodeInteger(1))
			} else {
				fmt.Fprint(conn, EncodeInteger(0))
			}

		default:
			fmt.Fprint(conn, EncodeError(fmt.Sprintf("unknown command '%s'", cmd)))
		}
	}
}
