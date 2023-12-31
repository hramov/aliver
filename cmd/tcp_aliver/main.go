package main

import (
	"context"
	"github.com/hramov/aliver/internal/tcp_aliver"
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

	cfg := tcp_aliver.Config{}
	err := tcp_aliver.LoadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("cannot parse config file: %v\n", err)
	}

	appCtx := context.Background()

	ctx, cancel := signal.NotifyContext(appCtx, os.Interrupt, os.Kill)

	server, err := tcp_aliver.NewServer(cfg.InstanceID, cfg.PortTCP, cfg.Timeout)
	if err != nil {
		log.Fatalf("cannot instantiate server: %v\n", err)
	}

	aliverInstance, err := tcp_aliver.New(
		cfg.ClusterID,
		cfg.InstanceID,
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
		server)
	if err != nil {
		log.Fatalf("cannot get instance: %v\n", err)
	}

	go aliverInstance.Start(ctx)

	<-ctx.Done()
	cancel()

	os.Exit(0)
}
