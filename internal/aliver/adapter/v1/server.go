package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hramov/aliver/internal/aliver/instance"
	"log"
	"net"
	"strings"
	"time"
)

const (
	UDP = "udp4"
	TCP = "tcp4"
)

type Server struct {
	instanceId    int
	ip            net.IP
	mask          net.IPMask
	portTcp       int
	portUdp       int
	broadcast     net.IP
	broadcastAddr *net.UDPAddr
	timeout       time.Duration
	pc            net.PacketConn
}

type rawMessageType struct {
	Name    string `json:"name"`
	Content any    `json:"content"`
}

func NewServer(instanceId int, ip net.IP, mask net.IPMask, broadcast net.IP, portTcp, portUdp int, timeout time.Duration) (*Server, error) {
	addr, err := net.ResolveUDPAddr(UDP, broadcast.String()+fmt.Sprintf(":%d", portUdp))
	if err != nil {
		log.Printf("cannot resolve udp address: %v\n", err)
		return nil, err
	}

	return &Server{
		instanceId:    instanceId,
		ip:            ip.To4(),
		portTcp:       portTcp,
		portUdp:       portUdp,
		mask:          mask,
		broadcast:     broadcast,
		broadcastAddr: addr,
		timeout:       timeout,
	}, nil
}

func (s *Server) ServeUDP(ctx context.Context, resCh chan<- instance.Message, errCh chan<- error) {
	pc, err := net.ListenPacket(UDP, fmt.Sprintf(":%d", s.portUdp))
	if err != nil {
		errCh <- fmt.Errorf("cannot listen udp, returning: %v\n", err)
		return
	}

	s.pc = pc

	defer func(pc net.PacketConn) {
		err = pc.Close()
		if err != nil {
			log.Printf("cannot close udp listener: %v\n", err)
		}
	}(s.pc)

	var messageName string
	var message any
	var addr net.Addr
	var n int

	log.Printf("UDP server started on %d\n", s.portUdp)
	buf := make([]byte, 128)

	for {
		select {
		case <-ctx.Done():
			return
		default:
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
}

func (s *Server) ServeTCP(ctx context.Context, resCh chan<- instance.Message, errCh chan<- error) {
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

}

func (s *Server) Health(ctx context.Context, errCh chan<- error) {
	for {
		if s.pc == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		select {
		case <-ctx.Done():
			return
		default:
			message := instance.IAM{
				InstanceID: s.instanceId,
				Ip:         s.ip,
			}

			err := s.SendBroadcast(instance.IAMMessage, message)
			if err != nil {
				errCh <- fmt.Errorf("cannot send message: %v\n", err)
			}

		}
		time.Sleep(s.timeout)
	}
}

func (s *Server) SendBroadcast(name string, content any) error {
	message := rawMessageType{
		Name:    name,
		Content: content,
	}
	rawMsgBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("cannot marshal raw message: %v\n", err)
		return fmt.Errorf("cannot marshal raw message: %v\n", err)
	}

	_, err = s.pc.WriteTo(rawMsgBytes, s.broadcastAddr)
	if err != nil {
		log.Printf("cannot write to pc: %v\n", err)
		return fmt.Errorf("cannot write to pc: %v\n", err)
	}

	return nil
}

func (s *Server) Send(ctx context.Context, conn net.Conn, name string, content any) error {
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
