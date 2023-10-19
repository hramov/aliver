package tcp_aliver

type IAM struct {
	InstanceID int    `json:"instance_id"`
	Mode       string `json:"mode"`
	Weight     int    `json:"weight"`
}
