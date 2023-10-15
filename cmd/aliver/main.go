package main

import (
	"context"
	v1 "github.com/hramov/aliver/internal/adapter/v1"
	"github.com/hramov/aliver/internal/config"
	"github.com/hramov/aliver/internal/fsm"
	"github.com/hramov/aliver/internal/instance"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
)

func main() {
	if os.Getenv("ENV") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("error loading .env file")
		}
	}

	configPath := os.Getenv("CONFIG_PATH")
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

	finiteStateMachine, currentStep := fsm.NewFsm()

	client := v1.NewClient(cfg.App.Timeout)

	server := v1.NewServer(cfg.App.Port)

	aliverInstance, err := instance.New(
		cfg.App.ClusterID,
		cfg.App.InstanceID,
		cfg.App.Ip,
		cfg.App.Mode,
		cfg.App.Weight,
		cfg.App.CheckScript,
		cfg.App.CheckInterval,
		cfg.App.CheckRetries,
		cfg.App.CheckTimeout,
		cfg.App.RunScript,
		cfg.App.RunTimeout,
		cfg.App.StopScript,
		cfg.App.StopTimeout,
		finiteStateMachine,
		currentStep,
		client,
		server)
	if err != nil {
		log.Fatalf("cannot get instance: %v\n", err)
	}

	go aliverInstance.Start(ctx)

	<-ctx.Done()
	cancel()

	os.Exit(0)
}
