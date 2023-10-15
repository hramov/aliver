package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hramov/aliver/internal/instance"
	"log"
	"net"
	"time"
)

const (
	UDP = "udp4"
)

type Server struct {
	instanceId int
	ip         net.IP
	portTcp    int
	portUdp    int
	broadcast  net.IP
	timeout    time.Duration
}

type rawMessageType struct {
	Name    string `json:"name"`
	Content any    `json:"content"`
}

func NewServer(instanceId, portTcp, portUdp int, ip net.IP, broadcast net.IP, timeout time.Duration) *Server {
	return &Server{
		instanceId: instanceId,
		ip:         ip,
		portTcp:    portTcp,
		portUdp:    portUdp,
		broadcast:  broadcast,
		timeout:    timeout,
	}
}

func (s *Server) ServeTCP(ctx context.Context, resCh chan<- instance.Message, errCh chan<- error) {
	var ln net.Listener
	var conn net.Conn
	var err error

	ln, err = net.Listen("tcp", fmt.Sprintf(":%d", s.portTcp))
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
		messageName, rawMessage, err = s.parse(buf)
		if err != nil {
			errCh <- fmt.Errorf("cannot parse message body: %v", err)
			continue
		}

		if messageName == instance.IAMMessage {
			resCh <- instance.Message{
				Name:    messageName,
				Content: rawMessage,
				Conn:    conn,
			}
		} else {
			resCh <- instance.Message{
				Name:    messageName,
				Content: rawMessage,
			}
		}

	}

}

func (s *Server) ServeUDP(ctx context.Context, resCh chan<- instance.Message, errCh chan<- error) {
	pc, err := net.ListenPacket(UDP, fmt.Sprintf(":%d", s.portUdp))
	if err != nil {
		panic(err)
	}
	defer func(pc net.PacketConn) {
		err = pc.Close()
		if err != nil {
			log.Printf("cannot close listener: %v\n", err)
		}
	}(pc)

	go s.health(ctx, pc)

	var messageName string
	var message any

	for {
		buf := make([]byte, 4096)
		_, _, err = pc.ReadFrom(buf)
		if err != nil {
			log.Printf("cannot read from listener: %v\n", err)
			continue
		}

		messageName, message, err = s.parse(buf)
		if err != nil {
			errCh <- err
			continue
		}

		resCh <- instance.Message{
			Name:    messageName,
			Content: message,
		}
	}
}

func (s *Server) parse(body []byte) (string, any, error) {
	message := rawMessageType{}
	err := json.Unmarshal(body, &message)
	if err != nil {
		return "", nil, err
	}
	return message.Name, message.Content, nil
}

func (s *Server) health(ctx context.Context, pc net.PacketConn) {
	for {
		time.Sleep(s.timeout)
		select {
		case <-ctx.Done():
			return
		default:
			addr, err := net.ResolveUDPAddr(UDP, s.broadcast.String())
			if err != nil {
				panic(err)
			}

			message := instance.IAM{
				InstanceID: s.instanceId,
				Ip:         s.ip,
			}

			rawMsg := rawMessageType{
				Name:    instance.IAMMessage,
				Content: message,
			}

			rawMsgBytes, err := json.Marshal(rawMsg)
			if err != nil {
				log.Printf("cannot marshal raw message: %v\n", err)
				continue
			}

			_, err = pc.WriteTo(rawMsgBytes, addr)
			if err != nil {
				log.Printf("cannot write to pc: %v\n", err)
			}
		}
	}
}
