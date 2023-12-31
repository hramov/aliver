package instance

import (
	"context"
	"github.com/hramov/aliver/internal/aliver/executor"
	"log"
	"math/rand"
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
	Conn    net.Conn
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

	quorum        int
	currentQuorum int
	votes         int
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

		quorum: 1,
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
	go i.checkService(ctx)
	go i.server.Health(ctx, i.errCh)

	taCh := time.NewTicker(2 * i.timeout)

	go func() {
		defer taCh.Stop()
		for _ = range taCh.C {
			if !i.leaderChosen && i.currentStep.Id != Election {
				i.chooseLeader(ctx)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			i.Stop()
			return
		case e := <-i.errCh:
			log.Println(e)
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
	case ELCMessage:
		messageMap := msg.Content.(map[string]any)
		message := ELC{
			InstanceID: int(messageMap["instance_id"].(float64)),
		}
		if message.InstanceID == i.instanceID {
			return
		}
		i.handleELCMessage(ctx, message)
		break
	case ELCACKMessage:
		messageMap := msg.Content.(map[string]any)
		message := ELCACK{
			InstanceID: int(messageMap["instance_id"].(float64)),
		}
		if message.InstanceID == i.instanceID {
			return
		}
		i.handleELCACKMessage(ctx, message)
		break
	case CFGMessage:
		messageMap := msg.Content.(map[string]any)
		message := CFG{
			InstanceID: int(messageMap["instance_id"].(float64)),
			Mode:       messageMap["mode"].(string),
			Weight:     int(messageMap["instance_id"].(float64)),
		}
		if message.InstanceID == i.instanceID {
			return
		}
		i.handleCFGMessage(ctx, message)
		break
	case CFGACKMessage:
		messageMap := msg.Content.(map[string]any)
		message := CFGACK{
			InstanceID:       int(messageMap["instance_id"].(float64)),
			ChosenInstanceID: int(messageMap["chosen_instance_id"].(float64)),
		}
		i.handleCFGACKMessage(ctx, message)
		break
	case LDRMessage:
		messageMap := msg.Content.(map[string]any)
		message := LDR{
			InstanceID: int(messageMap["instance_id"].(float64)),
		}
		i.handleLDRMessage(ctx, message)
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

		i.quorum++
	} else {
		Table[msg.InstanceID].lastActive = time.Now()
	}

	mode := UNKNOWN

	if i.currentStep.Id > Election {
		mode = i.mode
	}

	reply := ACK{
		InstanceID:       i.instanceID,
		RemoteInstanceID: msg.InstanceID,
		Ip:               i.ip,
		Mode:             mode,
		Weight:           i.weight,
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
			mode:       msg.Mode,
			weight:     msg.Weight,
			lastActive: time.Now(),
		}
		i.quorum++
	} else {
		Table[msg.InstanceID].lastActive = time.Now()
	}

	if i.mode == LEADER && msg.Mode == i.mode && msg.Weight < i.weight {
		i.chooseLeader(ctx)
		return
	}

	if msg.Mode == LEADER {
		i.leaderChosen = true
	}
}

func (i *Instance) handleELCMessage(ctx context.Context, msg ELC) {
	if i.currentStep.Id == Election || (!i.leaderChosen && i.currentStep.Id > Election) {
		return
	}

	i.currentQuorum = 0

	err := i.state.Transit(i.currentStep, Election)
	if err != nil {
		log.Printf("cannot transit to election: %v\n", err)
		return
	}
	log.Printf("change status to %s\n", i.currentStep.Title)

	elcAckMessage := &ELCACK{
		InstanceID: i.instanceID,
	}

	err = i.server.SendBroadcast(ELCACKMessage, elcAckMessage)
	if err != nil {
		log.Printf("cannot send message: %v\n", err)
		return
	}
}

func (i *Instance) handleELCACKMessage(ctx context.Context, msg ELCACK) {
	i.currentQuorum++

	if i.quorum == i.currentQuorum+1 {
		cfgMessage := &CFG{
			InstanceID: i.instanceID,
			Mode:       i.mode,
			Weight:     i.weight,
		}

		err := i.server.SendBroadcast(CFGMessage, cfgMessage)
		if err != nil {
			log.Printf("cannot send message: %v\n", err)
			return
		}
	}
}

func (i *Instance) handleCFGMessage(ctx context.Context, msg CFG) {
	i.currentQuorum = 0

	cfgAckMessage := &CFGACK{
		InstanceID: i.instanceID,
	}

	if msg.Mode == LEADER && i.mode != msg.Mode {
		cfgAckMessage.ChosenInstanceID = msg.InstanceID
		err := i.state.Transit(i.currentStep, Follower)
		if err != nil {
			log.Printf("cannot transit to follower: %v\n", err)
			return
		}
		log.Printf("change status to %s\n", i.currentStep.Title)
	} else if msg.Mode == FOLLOWER && i.mode != msg.Mode {
		cfgAckMessage.ChosenInstanceID = i.instanceID
		err := i.state.Transit(i.currentStep, PreLeader)
		if err != nil {
			log.Printf("cannot transit to PreLeader: %v\n", err)
			return
		}
		log.Printf("change status to %s\n", i.currentStep.Title)
	} else if msg.Mode == i.mode {
		cfgAckMessage.ChosenInstanceID = i.checkWeight(msg)
		if cfgAckMessage.ChosenInstanceID == i.instanceID {
			err := i.state.Transit(i.currentStep, PreLeader)
			if err != nil {
				log.Printf("cannot transit to preleader: %v\n", err)
				return
			}
			log.Printf("change status to %s\n", i.currentStep.Title)
		} else {
			err := i.state.Transit(i.currentStep, Follower)
			if err != nil {
				log.Printf("cannot transit to follower: %v\n", err)
				return
			}
			log.Printf("change status to %s\n", i.currentStep.Title)
		}
	}

	err := i.server.SendBroadcast(CFGACKMessage, cfgAckMessage)
	if err != nil {
		log.Printf("cannot send message: %v\n", err)
		return
	}
}

func (i *Instance) handleCFGACKMessage(ctx context.Context, msg CFGACK) {
	if i.currentStep.Id < PreLeader {
		return
	}

	i.currentQuorum++

	if msg.ChosenInstanceID == i.instanceID {
		i.votes++
	}

	if i.currentQuorum == i.quorum {
		if i.votes >= i.quorum/2+1 {
			err := i.state.Transit(i.currentStep, Leader)
			if err != nil {
				log.Printf("cannot transit to leader (handleCFGACKMessage): %v\n", err)
				return
			}
			log.Printf("change status to %s\n", i.currentStep.Title)

			err = i.server.SendBroadcast(LDRMessage, LDR{
				InstanceID: i.instanceID,
			})
			if err != nil {
				log.Printf("cannot send message: %v\n", err)
				return
			}
		} else {
			err := i.state.Transit(i.currentStep, Follower)
			if err != nil {
				log.Printf("cannot transit to follower: %v\n", err)
				return
			}
			log.Printf("change status to %s\n", i.currentStep.Title)
		}
	}
}

func (i *Instance) handleLDRMessage(ctx context.Context, message LDR) {
	i.leaderChosen = true
}

func (i *Instance) checkWeight(msg CFG) int {
	if msg.Weight > i.weight {
		return msg.InstanceID
	} else if msg.Weight < i.weight {
		return i.instanceID
	} else {
		if rand.Int()%2 == 0 {
			return msg.InstanceID
		} else {
			return i.instanceID
		}
	}
}

func (i *Instance) chooseLeader(ctx context.Context) {
	elcMessage := ELC{
		InstanceID: i.instanceID,
	}
	err := i.server.SendBroadcast(ELCMessage, elcMessage)
	if err != nil {
		log.Printf("cannot send ELC message: %v\n", err)
		return
	}
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

func (i *Instance) checkService(ctx context.Context) {
	var err error
	var cancel context.CancelFunc
	var timeoutCtx context.Context

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
			timeoutCtx, cancel = context.WithTimeout(ctx, i.checkTimeout)
			err = executor.Execute(timeoutCtx, i.checkScript)
			if err != nil {
				i.checkCh <- false
			} else {
				i.checkCh <- true
			}
		}
		time.Sleep(i.checkInterval)
	}
}

func (i *Instance) startService(ctx context.Context) {
	timeoutCtx, cancel := context.WithTimeout(ctx, i.runTimeout)
	defer cancel()

	err := executor.Execute(timeoutCtx, i.runScript)
	if err != nil {
		i.checkCh <- false
	} else {
		i.checkCh <- true
	}
}

func (i *Instance) stopService(ctx context.Context) {
	timeoutCtx, cancel := context.WithTimeout(ctx, i.stopTimeout)
	defer cancel()

	err := executor.Execute(timeoutCtx, i.stopScript)
	if err != nil {
		i.checkCh <- false
	} else {
		i.checkCh <- true
	}
}
