package v1

import (
	"context"
	"net"
	"sync"
	"time"
)

type Client struct {
	timeout time.Duration
}

var ClientInstance *Client
var once sync.Once

func NewClient(timeout time.Duration) *Client {
	once.Do(func() {
		ClientInstance = &Client{
			timeout: timeout,
		}
	})
	return ClientInstance
}

func (c *Client) SendIAM(ctx context.Context, instanceID int, ip net.IP) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) SendAck(ctx context.Context, instanceID int, mode string) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) SendCFG(ctx context.Context, instanceID int, mode string, weight int) error {
	//TODO implement me
	panic("implement me")
}

func (c *Client) SendELC(ctx context.Context, instanceID int) error {
	//TODO implement me
	panic("implement me")
}
