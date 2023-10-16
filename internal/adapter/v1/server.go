package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hramov/aliver/internal/instance"
	"log"
	"net"
	"strings"
	"time"
)

const (
	UDP = "udp4"
	TCP = "tcp"
)

type Server struct {
	instanceId int
	ip         net.IP
	mask       net.IPMask
	portTcp    int
	portUdp    int
	broadcast  net.IP
	timeout    time.Duration
}

type rawMessageType struct {
	Name    string `json:"name"`
	Content any    `json:"content"`
}

func NewServer(instanceId int, ip net.IP, mask net.IPMask, broadcast net.IP, portTcp, portUdp int, timeout time.Duration) *Server {

	return &Server{
		instanceId: instanceId,
		ip:         ip.To4(),
		portTcp:    portTcp,
		portUdp:    portUdp,
		mask:       mask,
		broadcast:  broadcast,
		timeout:    timeout,
	}
}

func (s *Server) GetUDPPort() int {
	return s.portUdp
}

func (s *Server) GetTCPPort() int {
	return s.portTcp
}

func (s *Server) ServeTCP(ctx context.Context, resCh chan<- instance.Message, errCh chan<- error) {
	var ln net.Listener
	var conn net.Conn
	var err error

	ln, err = net.Listen(TCP, fmt.Sprintf(":%d", s.portTcp))
	if err != nil {
		errCh <- err
		return
	}

	log.Printf("TCP server started on %d\n", s.portTcp)

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
	pc, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", s.portUdp))
	if err != nil {
		panic(err)
	}
	defer func(pc net.PacketConn) {
		err = pc.Close()
		if err != nil {
			log.Printf("cannot close udp listener: %v\n", err)
		}
	}(pc)

	go s.health(ctx, pc)

	var messageName string
	var message any
	var addr net.Addr
	var n int

	log.Printf("UDP server started on %d\n", s.portUdp)
	buf := make([]byte, 128)

	for {
		n, addr, err = pc.ReadFrom(buf)

		if err != nil {
			errCh <- fmt.Errorf("cannot read from listener: %v\n", err)
			continue
		}

		messageName, message, err = s.parse(buf[:n])
		if err != nil {
			errCh <- fmt.Errorf("cannot parse message body: %v\n", err)
			continue
		}

		resCh <- instance.Message{
			Name:    messageName,
			Content: message,
			Ip:      net.ParseIP(strings.Split(addr.String(), ":")[0]),
		}
	}
}

func (s *Server) parse(body []byte) (string, any, error) {
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

func (s *Server) health(ctx context.Context, pc net.PacketConn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
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

			addr, err := net.ResolveUDPAddr("udp4", s.broadcast.String()+fmt.Sprintf(":%d", s.portUdp))
			if err != nil {
				log.Printf("cannot resolve udp address: %v\n", err)
				continue
			}

			_, err = pc.WriteTo(rawMsgBytes, addr)
			if err != nil {
				log.Printf("cannot write to pc: %v\n", err)
				continue
			}

		}
		time.Sleep(s.timeout)
	}
}
