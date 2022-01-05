# Confluent Metric Worker

Worker service that can scrape metrics from Confluent Cloud and sync them to Google Cloud Monitoring

# Getting Started

- Clone repo
- Install Golang [Download](https://golang.org/dl)
- Install dependencies: `go get`
- Create local conf.yaml file: `cp conf.yaml.example conf.yaml`
- Update conf file with correct configuration. Ensure service account used for Google application credentials has sufficient permissions to use the Monitoring API.
- Run worker: `make worker`

# Background

The worker is comprised of two components: `Server`, and `Scraper`. When the worker process starts, it forks a new process for each of the components mentioned and manages a graceful shutdown of all components. Each component in described in detail below. On startup, the worker process will check if it needs to create custom metrics on Google Cloud Monitoring for each of the metrics defined in the config file.

## Server

The `Server` component is a simple HTTP server that should be used for monitoring and observability. An HTTP endpoint can be added to respond to health check probes to ensure that this worker is up and running. If desired, a Prometheus HTTP handler can be added to export metrics.

## Scraper

The `Scraper` component is the central brain of the worker. This process has an internal ticker that scrapes metrics from Confluent Cloud every minute.

## Config

The `Config` allows the user to specify the metrics and the corresponding labels they wish to scrape. The user must also specify a suffix for each of these metric filters so that they each get a unique metric type in Google Cloud Monitoring.
