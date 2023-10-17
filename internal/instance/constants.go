package instance

const (
	LEADER   = "leader"
	FOLLOWER = "follower"
	UNKNOWN  = "unknown"
)

const (
	Off          = -1
	Undiscovered = 1
	Discovered   = 2
	Election     = 3
	PreLeader    = 4
	Leader       = 5
	Follower     = 6
)

const (
	IAMMessage    = "IAM"
	CFGMessage    = "CFG"
	ACKMessage    = "ACK"
	ELCMessage    = "ELC"
	CFGACKMessage = "CFGACK"
)
