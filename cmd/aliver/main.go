package main

import (
	"context"
	v1 "github.com/hramov/aliver/internal/adapter/v1"
	"github.com/hramov/aliver/internal/config"
	"github.com/hramov/aliver/internal/instance"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
)

func main() {
	if os.Getenv("ALIVER_ENV") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("error loading .env file")
		}
	}

	configPath := os.Getenv("ALIVER_CONFIG_PATH")
	if configPath == "" {
		log.Fatal("config path env is not set")
	}

	cfg := config.Config{}
	err := config.LoadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("cannot parse config file: %v\n", err)
	}

	appCtx := context.Background()

	ctx, cancel := signal.NotifyContext(appCtx, os.Interrupt, os.Kill)

	finiteStateMachine, currentStep := instance.NewFsm()

	server, err := v1.NewServer(cfg.InstanceID, cfg.Ip, cfg.Mask, cfg.Broadcast, cfg.PortTCP, cfg.PortUDP, cfg.Timeout)
	if err != nil {
		log.Fatalf("cannot instantiate server: %v\n", err)
	}

	aliverInstance, err := instance.New(
		cfg.ClusterID,
		cfg.InstanceID,
		cfg.Ip,
		cfg.PortTCP,
		cfg.Mode,
		cfg.Weight,
		cfg.Timeout,
		cfg.CheckScript,
		cfg.CheckInterval,
		cfg.CheckRetries,
		cfg.CheckTimeout,
		cfg.RunScript,
		cfg.RunTimeout,
		cfg.StopScript,
		cfg.StopTimeout,
		finiteStateMachine,
		currentStep,
		server)
	if err != nil {
		log.Fatalf("cannot get instance: %v\n", err)
	}

	go aliverInstance.Start(ctx)

	<-ctx.Done()
	cancel()

	os.Exit(0)
}
