package instance

import (
	"context"
	"log"
	"net"
	"time"
)

type Server interface {
	ServeUDP(ctx context.Context, resCh chan<- Message, errCh chan<- error)
	Health(ctx context.Context, errCh chan<- error)
	SendBroadcast(name string, message any) error
}

type Message struct {
	Name    string
	Content any
	Ip      net.IP
}

type TableInstance struct {
	clusterID  string
	instanceID int
	ip         net.IP
	mode       string
	weight     int
	lastActive time.Time
}

type Instance struct {
	clusterID  string
	instanceID int
	ip         net.IP
	tcpPort    int
	mode       string
	weight     int
	timeout    time.Duration

	leaderChosen bool

	currentWeight int
	lastActive    time.Time

	checkScript   string
	checkInterval time.Duration
	checkRetries  int
	checkTimeout  time.Duration

	runScript  string
	runTimeout time.Duration

	stopScript  string
	stopTimeout time.Duration

	state       *Fsm
	currentStep *Step

	resCh   chan Message
	cmdCh   chan string
	checkCh chan bool
	errCh   chan error
	server  Server
}

var Table = make(map[int]*TableInstance)

func New(
	clusterId string, id int, ip net.IP, tcpPort int, mode string, weight int, timeout time.Duration,
	checkScript string, checkInterval time.Duration, checkRetries int, checkTimeout time.Duration,
	runScript string, runTimeout time.Duration,
	stopScript string, stopTimeout time.Duration,
	fsm *Fsm, currentStep *Step,
	server Server,
) (*Instance, error) {
	inst := &Instance{
		clusterID:     clusterId,
		instanceID:    id,
		ip:            ip,
		tcpPort:       tcpPort,
		mode:          mode,
		weight:        weight,
		currentWeight: weight,
		timeout:       timeout,

		checkScript:   checkScript,
		checkInterval: checkInterval,
		checkRetries:  checkRetries,
		checkTimeout:  checkTimeout,
		runScript:     runScript,
		runTimeout:    runTimeout,

		stopScript:  stopScript,
		stopTimeout: stopTimeout,

		state:       fsm,
		currentStep: currentStep,
		resCh:       make(chan Message, 5),
		cmdCh:       make(chan string),
		checkCh:     make(chan bool),
		errCh:       make(chan error),
		server:      server,
	}

	Table[id] = &TableInstance{
		clusterID:  clusterId,
		instanceID: id,
		ip:         ip,
		mode:       mode,
		weight:     weight,
		lastActive: time.Now(),
	}

	return inst, nil
}

func (i *Instance) Start(ctx context.Context) {
	go i.server.ServeUDP(ctx, i.resCh, i.errCh)
	go i.handleMessages(ctx)
	go i.checkService()
	go i.server.Health(ctx, i.errCh)

	taCh := time.After(3 * i.timeout)

	for {
		select {
		case <-ctx.Done():
			i.Stop()
			return
		case e := <-i.errCh:
			log.Println(e)
		case <-taCh:
			if !i.leaderChosen {
				i.chooseLeader(ctx)
			}
		}
	}
}

func (i *Instance) Stop() {
	// logic for correctly exit the process
}

func (i *Instance) handleMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-i.resCh:
			i.handle(ctx, msg)
		}
	}
}

func (i *Instance) handle(ctx context.Context, msg Message) {
	switch msg.Name {
	case IAMMessage:
		messageMap := msg.Content.(map[string]any)
		message := IAM{
			InstanceID: int(messageMap["instance_id"].(float64)),
			Ip:         msg.Ip,
		}
		if message.InstanceID == i.instanceID {
			return
		}
		i.handleIAMMessage(ctx, message)
		break
	case ACKMessage:
		messageMap := msg.Content.(map[string]any)
		message := ACK{
			InstanceID:       int(messageMap["instance_id"].(float64)),
			RemoteInstanceID: int(messageMap["remote_instance_id"].(float64)),
			Ip:               msg.Ip,
		}
		if message.RemoteInstanceID == message.InstanceID {
			return
		}
		i.handleACKMessage(ctx, message)
		break
	case CFGMessage:
		message := msg.Content.(CFG)
		i.handleCFGMessage(message)
		break
	case ELCMessage:
		message := msg.Content.(ELC)
		i.handleELCMessage(message)
		break
	default:
		log.Printf("unknown message: %v\n", msg.Name)
	}
}

func (i *Instance) handleIAMMessage(ctx context.Context, msg IAM) {
	var err error
	if _, ok := Table[msg.InstanceID]; !ok {
		Table[msg.InstanceID] = &TableInstance{
			clusterID:  i.clusterID,
			instanceID: msg.InstanceID,
			ip:         msg.Ip,
			mode:       UNKNOWN,
			weight:     0,
			lastActive: time.Now(),
		}
	} else {
		Table[msg.InstanceID].lastActive = time.Now()
	}

	reply := ACK{
		InstanceID:       i.instanceID,
		RemoteInstanceID: msg.InstanceID,
		Ip:               i.ip,
	}

	err = i.server.SendBroadcast(ACKMessage, reply)
	if err != nil {
		log.Printf("cannot send message: %v\n", err)
		return
	}
}

func (i *Instance) handleACKMessage(ctx context.Context, msg ACK) {
	if i.currentStep.Id == Undiscovered {
		err := i.state.Transit(i.currentStep, Discovered)
		if err != nil {
			log.Printf("cannot transit: %v\n", err)
			return
		}
		log.Printf("change status to %s\n", i.currentStep.Title)
	}

	if _, ok := Table[msg.InstanceID]; !ok {
		Table[msg.InstanceID] = &TableInstance{
			clusterID:  i.clusterID,
			instanceID: msg.InstanceID,
			ip:         msg.Ip,
			mode:       UNKNOWN,
			weight:     0,
			lastActive: time.Now(),
		}
	} else {
		Table[msg.InstanceID].lastActive = time.Now()
	}
}

func (i *Instance) handleCFGMessage(msg CFG) {

}

func (i *Instance) handleELCMessage(msg ELC) {

}

func (i *Instance) chooseLeader(ctx context.Context) {
	//for _, v := range Table {
	//	err := i.client.SendCFG(ctx, v.instanceID, LEADER, i.currentWeight, v.conn)
	//	if err != nil {
	//		log.Printf("cannot send CFG message to %v: %v\n", v.conn.RemoteAddr(), err)
	//	}
	//}
}

func (i *Instance) checkInstances(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for _, inst := range Table {
				if time.Since(inst.lastActive) > 3*i.checkTimeout {
					log.Printf("instance %s is not active\n", inst.ip)
					if inst.mode == LEADER {
						log.Printf("instance %s was the leader\n", inst.ip)
						// TODO send ELC to all other instances to choose a new leader
					}
				}
			}
			time.Sleep(i.checkInterval)
		}
	}
}

func (i *Instance) checkService() {
	for {
		time.Sleep(i.checkInterval)
		i.checkCh <- true
	}
}
