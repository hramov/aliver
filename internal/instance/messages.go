package instance

import "net"

const (
	LEADER   = "leader"
	FOLLOWER = "follower"
	UNKNOWN  = "unknown"
)

const (
	InstanceOff  = -1
	Undiscovered = 1
	Discovered   = 2
	Election     = 3
	Leader       = 4
	Follower     = 5
)

const (
	IAMMessage = "IAM"
	CFGMessage = "CFG"
	ACKMessage = "ACK"
	ELCMessage = "ELC"
)

type IAM struct {
	InstanceID int    `json:"instance_id"`
	Ip         net.IP `json:"ip"`
}

type CFG struct {
	InstanceID int    `json:"instance_id"`
	Mode       string `json:"mode"`
	Weight     int    `json:"weight"`
}

type ACK struct {
	InstanceID       int    `json:"instance_id"`
	Ip               net.IP `json:"ip"`
	RemoteInstanceID int    `json:"remote_instance_id"`
}

type ELC struct {
	InstanceID int `json:"instance_id"`
}
