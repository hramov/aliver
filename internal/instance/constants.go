package instance

const (
	LEADER   = "leader"
	FOLLOWER = "follower"
	UNKNOWN  = "unknown"
)

const (
	Off          = 0
	Undiscovered = 1
	Discovered   = 2
	Election     = 3
	Follower     = 4
	PreLeader    = 5
	Leader       = 6
)

const (
	IAMMessage    = "IAM"
	CFGMessage    = "CFG"
	ACKMessage    = "ACK"
	ELCMessage    = "ELC"
	CFGACKMessage = "CFGACK"
)
