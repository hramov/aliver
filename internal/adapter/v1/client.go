package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hramov/aliver/internal/instance"
	"net"
	"sync"
)

type Client struct{}

var ClientInstance *Client

var once sync.Once

func NewClient() (*Client, error) {
	once.Do(func() {
		ClientInstance = &Client{}
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
	return fmt.Errorf("conn is nil")
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
