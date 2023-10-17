package instance

import "net"

type IAM struct {
	InstanceID int    `json:"instance_id"`
	Ip         net.IP `json:"ip"`
}

type ACK struct {
	InstanceID       int    `json:"instance_id"`
	Ip               net.IP `json:"ip"`
	RemoteInstanceID int    `json:"remote_instance_id"`
	Mode             string `json:"mode"`
	Weight           int    `json:"weight"`
}

type ELC struct {
	InstanceID int `json:"instance_id"`
}

type CFG struct {
	InstanceID int    `json:"instance_id"`
	Mode       string `json:"mode"`
	Weight     int    `json:"weight"`
}

type CFGACK struct {
	InstanceID       int `json:"instance_id"`
	ChosenInstanceID int `json:"chosen_instance_id"`
}
