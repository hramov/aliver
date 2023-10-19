package tcp_aliver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hramov/aliver/internal/aliver/instance"
	"log"
	"math"
	"net"
	"time"
)

const (
	TCP = "tcp4"
)

type server struct {
	instanceId int
	portTcp    int
	timeout    time.Duration
}

type rawMessageType struct {
	Name    string `json:"name"`
	Content any    `json:"content"`
}

func NewServer(instanceId int, portTcp int, timeout time.Duration) (Server, error) {
	return &server{
		instanceId: instanceId,
		portTcp:    portTcp,
		timeout:    timeout,
	}, nil
}

func (s *server) ServeTCP(ctx context.Context, resCh chan<- message, errCh chan<- error) {
	var ln net.Listener
	var conn net.Conn
	var err error

	ln, err = net.Listen(TCP, fmt.Sprintf(":%d", s.portTcp))
	if err != nil {
		errCh <- fmt.Errorf("cannot listen tcp, returning: %v\n", err)
		return
	}

	log.Printf("TCP server started on %d\n", s.portTcp)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err = ln.Accept()
			if err != nil {
				errCh <- err
				return
			}

			var buf []byte

			_, err = conn.Read(buf)
			if err != nil {
				errCh <- fmt.Errorf("cannot read from listener: %v\n", err)
				continue
			}

			var messageName string
			var rawMessage any

			messageName, rawMessage, err = s.parse(buf)
			if err != nil {
				errCh <- fmt.Errorf("cannot parse message body: %v", err)
				continue
			}

			resCh <- message{
				name:    messageName,
				content: rawMessage,
				conn:    conn,
			}
		}

	}

}

func (s *server) Health(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg := instance.IAM{
				InstanceID: s.instanceId,
			}

			err := s.Send(ctx, serverConn, IAMMessage, msg)
			if err != nil {
				errCh <- fmt.Errorf("cannot send message: %v\n", err)
			}

		}
		time.Sleep(s.timeout)
	}
}

func (s *server) Send(ctx context.Context, conn net.Conn, name string, content any) error {
	message := rawMessageType{
		Name:    name,
		Content: content,
	}
	rawMsgBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("cannot marshal raw message: %v\n", err)
		return fmt.Errorf("cannot marshal raw message: %v\n", err)
	}

	_, err = conn.Write(rawMsgBytes)
	if err != nil {
		log.Printf("cannot write to conn: %v\n", err)
		return fmt.Errorf("cannot write to conn: %v\n", err)
	}

	return nil
}

func (s *server) Connect(ctx context.Context, ip net.IP, port int) (net.Conn, error) {
	counter := 0
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			conn, err := net.Dial(TCP, fmt.Sprintf("%s:%d", ip.String(), port))
			if err == nil {
				if counter > 10 {
					return nil, err
				}
				counter++
				time.Sleep(100 * time.Millisecond * time.Duration(math.Pow(float64(2), float64(counter))))
			}

			msg := instance.IAM{
				InstanceID: s.instanceId,
			}

			err = s.Send(ctx, serverConn, IAMMessage, msg)
			if err != nil {
				return conn, fmt.Errorf("cannot send message: %v\n", err)
			}

			return conn, nil
		}
	}
}

func (s *server) parse(body []byte) (string, any, error) {
	if len(body) == 0 {
		return "", nil, fmt.Errorf("empty body")
	}

	message := rawMessageType{}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return "", nil, err
	}
	return message.Name, message.Content, nil
}
