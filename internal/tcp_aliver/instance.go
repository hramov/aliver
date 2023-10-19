package tcp_aliver

import (
	"context"
	"github.com/hramov/aliver/internal/aliver/executor"
	"log"
	"net"
	"time"
)

type Server interface {
	ServeTCP(ctx context.Context, resCh chan<- message, errCh chan<- error)
	Health(ctx context.Context, errCh chan<- error)
	Send(ctx context.Context, conn net.Conn, name string, content any) error
	Connect(ctx context.Context, ip net.IP, port int) (net.Conn, error)
}

type Instance struct {
	clusterID  string
	instanceID int
	ip         net.IP
	tcpPort    int
	mode       string
	weight     int
	timeout    time.Duration

	currentMode   string
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

	resCh   chan message
	cmdCh   chan string
	checkCh chan bool
	errCh   chan error
	server  Server
}

type message struct {
	name    string
	content any
	ip      net.IP
	conn    net.Conn
}

var serverConn net.Conn
var serverId int

func New(
	clusterId string, id int, tcpPort int, mode string, weight int, timeout time.Duration,
	checkScript string, checkInterval time.Duration, checkRetries int, checkTimeout time.Duration,
	runScript string, runTimeout time.Duration,
	stopScript string, stopTimeout time.Duration,
	server Server,
) (*Instance, error) {
	inst := &Instance{
		clusterID:     clusterId,
		instanceID:    id,
		tcpPort:       tcpPort,
		mode:          mode,
		weight:        weight,
		currentMode:   mode,
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

		resCh:   make(chan message, 5),
		cmdCh:   make(chan string),
		checkCh: make(chan bool),
		errCh:   make(chan error),
		server:  server,
	}

	return inst, nil
}

func (i *Instance) Start(ctx context.Context) {
	go i.server.ServeTCP(ctx, i.resCh, i.errCh)
	go i.handleMessages(ctx)

	conn, err := i.server.Connect(ctx, i.ip, i.tcpPort)
	if err != nil || i.mode == LEADER {
		log.Printf("cannot connect to second server: %v\n", err)
		i.currentMode = LEADER
		i.startService(ctx)
	} else {
		i.stopService(ctx)
	}

	serverConn = conn

	go i.checkService(ctx)
	go i.server.Health(ctx, i.errCh)

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

func (i *Instance) handle(ctx context.Context, msg message) {
	switch msg.name {
	case IAMMessage:
		messageMap := msg.content.(map[string]any)
		m := IAM{
			InstanceID: int(messageMap["instance_id"].(float64)),
		}
		if m.InstanceID == i.instanceID {
			return
		}

		if serverId == 0 {
			serverId = m.InstanceID
		}

		if serverConn == nil {
			serverConn = msg.conn
		}

		if m.Mode == LEADER && i.mode != m.Mode {
			i.currentMode = FOLLOWER
			i.stopService(ctx)
			log.Printf("change status to %s\n", m.Mode)
		} else if m.Mode == FOLLOWER && i.mode != m.Mode {
			i.currentMode = LEADER
			i.startService(ctx)
			log.Printf("change status to %s\n", m.Mode)
		} else if m.Mode == i.mode {
			if m.Weight > i.weight {
				i.currentMode = FOLLOWER
				i.stopService(ctx)
				log.Printf("change status to %s\n", m.Mode)
			} else {
				i.currentMode = LEADER
				i.startService(ctx)
				log.Printf("change status to %s\n", m.Mode)
			}
		}
		break
	}
}

func (i *Instance) checkInstances(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			//for _, inst := range Table {
			//	if time.Since(inst.lastActive) > 3*i.checkTimeout {
			//		log.Printf("instance %s is not active\n", inst.ip)
			//		if inst.mode == LEADER {
			//			log.Printf("instance %s was the leader\n", inst.ip)
			//			// TODO send ELC to all other instances to choose a new leader
			//		}
			//	}
			//}
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
			if i.mode == LEADER {
				timeoutCtx, cancel = context.WithTimeout(ctx, i.checkTimeout)
				err = executor.Execute(timeoutCtx, i.checkScript)
				if err != nil {
					i.checkCh <- false
				} else {
					i.checkCh <- true
				}
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
