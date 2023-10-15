package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hramov/aliver/internal/instance"
	"net"
)

type Server struct {
	port int
}

type rawMessageType struct {
	Name    string `json:"name"`
	Content any    `json:"content"`
}

func NewServer(port int) *Server {
	return &Server{
		port: port,
	}
}

func (l *Server) Serve(ctx context.Context, resCh chan<- instance.Message, errCh chan<- error) {
	var ln net.Listener
	var conn net.Conn
	var err error

	ln, err = net.Listen("tcp", fmt.Sprintf(":%d", l.port))
	if err != nil {
		errCh <- err
		return
	}

	for {
		if ctx.Err() != nil {
			errCh <- ctx.Err()
			return
		}

		conn, err = ln.Accept()
		if err != nil {
			errCh <- err
			return
		}

		var buf []byte

		_, err = conn.Read(buf)
		if err != nil {
			errCh <- err
			continue
		}

		var messageName string
		var rawMessage any

		// rawMessage format: "{ \"name\": \"IAM\", \"content\": { \"instance_id\": 1, \"ip\": \"127.0.0.1\" } }"
		messageName, rawMessage, err = l.parse(buf)
		if err != nil {
			errCh <- fmt.Errorf("cannot parse message body: %v", err)
			continue
		}

		resCh <- instance.Message{
			Name:    messageName,
			Content: rawMessage,
		}

	}

}

func (l *Server) parse(body []byte) (string, any, error) {
	message := rawMessageType{}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return "", nil, err
	}
	return message.Name, message.Content, nil
}
