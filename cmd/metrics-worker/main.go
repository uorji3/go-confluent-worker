package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/uorji3/go-confluent-worker/app/config"
	"github.com/uorji3/go-confluent-worker/app/logger"
	"github.com/uorji3/go-confluent-worker/app/scraper"
	"github.com/uorji3/go-confluent-worker/app/server"
	"gopkg.in/yaml.v2"
)

var (
	BuildDate string
)

func main() {
	var configFilePath string

	flag.StringVar(&configFilePath, "config-file-path", "", "Config file path")
	flag.Parse()

	if configFilePath == "" {
		log.Fatal("Must provide config file path")
	}

	var config config.Config
	b, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatalf("error parsing config file: %v", err)
	}

	err = config.Validate()
	if err != nil {
		log.Fatalf("error validating config: %v", err)
	}

	if config.Environment.DisableStdOutLogger {
		os.Setenv("DISABLE_STDOUT_LOGGER", "true")
	}

	if config.Environment.EnableGCPLogger {
		os.Setenv("ENABLE_GCP_LOGGER", "true")
	}

	if config.Environment.Environment != "" {
		os.Setenv("ENVIRONMENT", config.Environment.Environment)
	}

	if config.Environment.GCPLoggerName != "" {
		os.Setenv("GCP_LOGGER_NAME", config.Environment.GCPLoggerName)
	}

	if config.Environment.GoogleApplicationCredentials != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.Environment.GoogleApplicationCredentials)
	}

	if config.Environment.SentryDSN != "" {
		os.Setenv("SENTRY_DSN", config.Environment.SentryDSN)
	}

	// Run after env has been loaded for GCP & Sentry loggers
	logger.Initialize()
	defer logger.Flush()

	ctx, cancel := context.WithCancel(context.Background())

	scraper, err := scraper.NewScraper(ctx, config)
	if err != nil {
		logger.Fatalf("Failed to initialize scraper client: %v", err)
	}

	server := server.NewServer(config.Environment.Port, BuildDate)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		if err := scraper.Run(ctx); err != nil {
			logger.Errorf("Failed to run scraper: %v", err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		if err := server.Run(ctx); err != nil {
			logger.Warnf("Failed to run server: %v", err)
		}
		wg.Done()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	oscall := <-quit
	logger.Infof("Received system call: %+v. Shutting Confluent metrics worker down...", oscall)
	cancel()

	wg.Wait()

	scraper.Close()

	logger.Info("Confluent metrics worker exiting")
}
