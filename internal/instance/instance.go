package instance

import (
	"context"
	"fmt"
	"github.com/hramov/aliver/internal/fsm"
	"log"
	"net"
	"time"
)

type HttpClient interface {
	SendIAM(ctx context.Context, instanceID int, ip net.IP, conn net.Conn) error
	SendAck(ctx context.Context, instanceID int, mode string, conn net.Conn) error
	SendCFG(ctx context.Context, instanceID int, mode string, weight int, conn net.Conn) error
	SendELC(ctx context.Context, instanceID int, weight int, conn net.Conn) error
}

type Server interface {
	ServeTCP(ctx context.Context, resCh chan<- Message, errCh chan<- error)
	ServeUDP(ctx context.Context, resCh chan<- Message, errCh chan<- error)
	GetUDPPort() int
}

type Message struct {
	Name    string
	Content any
	Conn    net.Conn
}

type Instance struct {
	clusterID     string
	instanceID    int
	ip            net.IP
	conn          net.Conn
	mode          string
	weight        int
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

	fsm         *fsm.Fsm
	currentStep *fsm.Step
	client      HttpClient

	resCh   chan Message
	cmdCh   chan string
	checkCh chan bool
	errCh   chan error
	server  Server
}

var InstanceTable = make(map[int]*Instance)

func New(
	clusterId string, id int, ip string, mode string, weight int,
	checkScript string, checkInterval time.Duration, checkRetries int, checkTimeout time.Duration,
	runScript string, runTimeout time.Duration,
	stopScript string, stopTimeout time.Duration,
	fsm *fsm.Fsm, currentStep *fsm.Step,
	client HttpClient,
	server Server,
) (*Instance, error) {
	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return nil, fmt.Errorf("ip is not correct")
	}

	inst := &Instance{
		clusterID:     clusterId,
		instanceID:    id,
		ip:            net.ParseIP(ip),
		mode:          mode,
		weight:        weight,
		currentWeight: weight,

		checkScript:   checkScript,
		checkInterval: checkInterval,
		checkRetries:  checkRetries,
		checkTimeout:  checkTimeout,
		runScript:     runScript,
		runTimeout:    runTimeout,

		stopScript:  stopScript,
		stopTimeout: stopTimeout,

		fsm:         fsm,
		currentStep: currentStep,
		client:      client,
		resCh:       make(chan Message),
		cmdCh:       make(chan string),
		checkCh:     make(chan bool),
		errCh:       make(chan error),
		server:      server,
	}

	InstanceTable[id] = inst

	return inst, nil
}

func (i *Instance) Start(ctx context.Context) {
	go i.server.ServeTCP(ctx, i.resCh, i.errCh)
	go i.server.ServeUDP(ctx, i.resCh, i.errCh)

	go i.check(i.checkInterval, i.checkScript, i.checkCh)

	go i.Handle(ctx, i.resCh)

	for {
		select {
		case <-ctx.Done():
			i.Stop()
			return
		case e := <-i.errCh:
			log.Println(e)
			break
		case c := <-i.checkCh:
			if !c {
				i.currentWeight = InstanceOff
				if i.currentStep.Id == Leader {
					err := i.client.SendELC(ctx, i.instanceID, i.currentWeight, nil)
					if err != nil {
						log.Printf("cannot send message: %v\n", err)
					}

					_, err = i.fsm.Transit(i.currentStep, 2)
					if err != nil {
						log.Printf("cannot transit fsm: %v\n", err)
					}
				}

			} else {
				i.currentWeight = i.weight
				if i.currentStep.Id == Undiscovered {
				}
			}
		}
	}
}

func (i *Instance) Handle(ctx context.Context, resCh <-chan Message) {
	select {
	case <-ctx.Done():
		return
	case msg := <-resCh:
		i.handle(ctx, msg)
	}
}

func (i *Instance) handle(ctx context.Context, msg Message) {
	switch msg.Name {
	case IAMMessage:
		messageMap := msg.Content.(map[string]any)
		message := IAM{
			InstanceID: int(messageMap["instance_id"].(float64)),
			Ip:         net.ParseIP(messageMap["ip"].(string)),
		}

		if message.InstanceID == i.instanceID {
			return
		}

		i.handleIAMMessage(ctx, message, msg.Conn)
		break
	case CFGMessage:
		message := msg.Content.(CFG)
		i.handleCFGMessage(message)
		break
	case ACKMessage:
		message := msg.Content.(ACK)
		i.handleACKMessage(message)
		break
	case ELCMessage:
		message := msg.Content.(ELC)
		i.handleELCMessage(message)
		break
	default:
		log.Printf("unknown message: %v\n", msg.Name)
	}
}

func (i *Instance) handleIAMMessage(ctx context.Context, msg IAM, conn net.Conn) {
	var err error

	log.Printf("found new instance: %s\n", msg.Ip)

	if conn == nil {
		// from UDP
		conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", msg.Ip.String(), i.server.GetUDPPort()))
		if err != nil {
			log.Printf("cannot dial tcp: %v\n", err)
		}

		err = i.client.SendIAM(ctx, i.instanceID, i.ip, conn)
		if err != nil {
			log.Printf("cannot send IAM message: %v\n", err)
		}
	}

	log.Printf("connected to new instance: %s\n", msg.Ip)

	if _, ok := InstanceTable[msg.InstanceID]; !ok {
		InstanceTable[msg.InstanceID] = &Instance{
			clusterID:  i.clusterID,
			instanceID: msg.InstanceID,
			ip:         msg.Ip,
			conn:       conn,
			mode:       UNKNOWN,
			weight:     0,
			lastActive: time.Now(),
		}
	} else {
		InstanceTable[msg.InstanceID].lastActive = time.Now()
	}

	InstanceTable[msg.InstanceID] = &Instance{
		clusterID:  i.clusterID,
		instanceID: msg.InstanceID,
		ip:         msg.Ip,
		conn:       conn,
		mode:       UNKNOWN,
		weight:     0,
	}
}

func (i *Instance) handleCFGMessage(msg CFG) {

}

func (i *Instance) handleACKMessage(msg ACK) {

}

func (i *Instance) handleELCMessage(msg ELC) {

}

func (i *Instance) checkInstances(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for _, inst := range InstanceTable {
				if time.Since(inst.lastActive) > 3*i.checkTimeout {
					log.Printf("instance %s is not active\n", inst.ip)
					if inst.mode == LEADER {
						log.Printf("instance %s was the leader, need to choose another one\n", inst.ip)
						// TODO send ELC to all other instances to choose a new leader
					}
				}
			}
			time.Sleep(i.checkInterval)
		}
	}
}

func (i *Instance) check(interval time.Duration, script string, check chan<- bool) {
	for {
		time.Sleep(interval)
		check <- true
	}
}

func (i *Instance) Stop() {
	// logic for correctly exit the process
}
