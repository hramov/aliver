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

func NewServer(instanceId, portTcp, portUdp int, ip net.IP, mask int, timeout time.Duration) *Server {
	m := net.CIDRMask(mask, 32)
	broadcast := net.ParseIP("0.0.0.0").To4()

	ip2 := net.ParseIP(ip.String()).To4()

	for i := 0; i < len(ip2); i++ {
		broadcast[i] = ip2[i] | ^m[i]
	}

	return &Server{
		instanceId: instanceId,
		ip:         ip2,
		portTcp:    portTcp,
		portUdp:    portUdp,
		mask:       m,
		broadcast:  broadcast,
		timeout:    timeout,
	}
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
	pc, err := net.ListenPacket(UDP, fmt.Sprintf(":%d", s.portUdp))
	if err != nil {
		log.Printf("cannot listen udp: %v\n", err)
	}
	defer func(pc net.PacketConn) {
		err = pc.Close()
		if err != nil {
			log.Printf("cannot close listener: %v\n", err)
		}
	}(pc)

	log.Printf("sending broadcast IAM message")
	go s.health(ctx, pc)

	var messageName string
	var message any
	var n int

	log.Printf("UDP server started on %d\n", s.portUdp)

	for {
		buf := make([]byte, 1024)

		n, _, err = pc.ReadFrom(buf)
		if err != nil {
			log.Printf("cannot read from listener: %v\n", err)
			continue
		}

		messageName, message, err = s.parse(buf[0:n])
		if err != nil {
			errCh <- fmt.Errorf("cannot parse message body: %v\n", err)
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
		select {
		case <-ctx.Done():
			return
		default:
			addr, err := net.ResolveUDPAddr(UDP, s.broadcast.String()+fmt.Sprintf(":%d", s.portUdp))
			if err != nil {
				log.Printf("cannot resolve udp address: %v\n", err)
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
		time.Sleep(s.timeout)
	}
}

func (s *Server) GetUDPPort() int {
	return s.portUdp
}
