package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/uorji3/go-confluent-worker/app/logger"
)

type Server struct {
	server *http.Server
}

func NewServer(port, rawBuildDate string) *Server {
	if port == "" {
		port = "8080"
	}

	var buildDate string
	unixTimestamp, err := strconv.ParseInt(rawBuildDate, 10, 64)
	if err != nil {
		buildDate = "N/A"
	} else {
		t := time.Unix(unixTimestamp, 0).In(time.UTC)

		buildDate = t.Format("Mon Jan 2 15:04:05 MST 2006")
	}

	info := map[string]string{
		"Info":       "Confluent Metrics Worker",
		"Build Date": buildDate,
		"Status":     "ok",
	}

	b, _ := json.MarshalIndent(info, "", "  ")

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, string(b))
		},
	))

	s := &Server{
		server: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
	}

	return s
}

func (s *Server) Run(ctx context.Context) error {
	logger.Infof("Server starting on port: %v", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to run server: %+v", err)
		}
	}()

	<-ctx.Done()

	logger.Info("Shutting server down gracefully...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer func() {
		cancel()
	}()

	err := s.server.Shutdown(ctxShutDown)
	if err != nil && err != http.ErrServerClosed {
		logger.Errorf("Server failed to shutdown gracefully: %+v", err)
		return err
	}

	logger.Info("Server gracefully terminated")

	return nil
}
