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
	go i.Handle(ctx, i.resCh)
	go i.Check(i.checkInterval, i.checkScript, i.checkCh)

	err := i.client.SendIAM(ctx, i.instanceID, i.ip, nil)
	if err != nil {
		log.Printf("cannot send iam message: %v\n", err)
	}

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
					err = i.client.SendELC(ctx, i.instanceID, i.currentWeight, nil)
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
		message := msg.Content.(IAM)
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
	log.Printf("found new instance: %s\n", msg.Ip)

	if conn == nil {
		log.Printf("conn is nil: %v\n", conn)
		return
	}

	InstanceTable[msg.InstanceID] = &Instance{
		clusterID:  i.clusterID,
		instanceID: msg.InstanceID,
		ip:         msg.Ip,
		conn:       conn,
		mode:       UNKNOWN,
		weight:     0,
	}

	err := i.client.SendIAM(ctx, i.instanceID, i.ip, conn)
	if err != nil {
		log.Printf("cannot send AIM message: %v\n", err)
	}
}

func (i *Instance) handleCFGMessage(msg CFG) {

}

func (i *Instance) handleACKMessage(msg ACK) {

}

func (i *Instance) handleELCMessage(msg ELC) {

}

func (i *Instance) Check(interval time.Duration, script string, check chan<- bool) {
	for {
		time.Sleep(interval)
		check <- true
	}
}

func (i *Instance) Stop() {
	// logic for correctly exit the process
}
