build:
	export BUILD_DATE=$$(date +%s) && GOOS=linux go build -ldflags "-X main.BuildDate=$$BUILD_DATE" -o confluentmetricsworker cmd/metrics-worker/main.go

worker:
	go run cmd/metrics-worker/main.go --config-file-path conf.yaml
