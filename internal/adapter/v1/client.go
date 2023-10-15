package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hramov/aliver/internal/instance"
	"net"
	"sync"
)

type Client struct {
	broadcast net.IP
}

var ClientInstance *Client
var once sync.Once

func NewClient(broadcastIp string) (*Client, error) {
	broadcast := net.ParseIP(broadcastIp)
	if broadcast == nil {
		return nil, fmt.Errorf("broadcast IP is nil")
	}

	once.Do(func() {
		ClientInstance = &Client{
			broadcast: net.ParseIP(broadcastIp),
		}
	})
	return ClientInstance, nil
}

func (c *Client) SendIAM(ctx context.Context, instanceID int, ip net.IP, conn net.Conn) error {
	if conn != nil {
		content := instance.IAM{
			InstanceID: instanceID,
			Ip:         ip,
		}

		rawMsg := rawMessageType{
			Name:    instance.IAMMessage,
			Content: content,
		}

		rawMsgBytes, err := json.Marshal(rawMsg)

		_, err = conn.Write(rawMsgBytes)
		return err
	}

	//TODO send broadcast message
	return nil
}

func (c *Client) SendAck(ctx context.Context, instanceID int, mode string, conn net.Conn) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) SendCFG(ctx context.Context, instanceID int, mode string, weight int, conn net.Conn) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) SendELC(ctx context.Context, instanceID int, weight int, conn net.Conn) error {
	//TODO implement me
	panic("implement me")
}
