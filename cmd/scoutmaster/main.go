package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/nylar/scout/health"
	"github.com/nylar/scout/health/healthpb"

	"github.com/nylar/scout/discovery/discoverypb"
	"github.com/nylar/scout/files/filepb"
	"github.com/nylar/scout/master"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
)

var errNodeNotRegistered = errors.New("node not registered")

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	httpPort := flag.Int("http", 3001, "HTTP server port")
	grpcPort := flag.Int("grpc", 3000, "gRPC server port")

	flag.Parse()

	server := grpc.NewServer()

	master := master.NewMaster()
	defer master.SetHealthStatus(health.NotServing)

	go master.HealthCheck()

	discoverypb.RegisterDiscoveryServiceServer(server, master)
	filepb.RegisterFileServiceServer(server, master)
	healthpb.RegisterHealthServer(server, master)

	http.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(master.Aggregate()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		Handler: http.DefaultServeMux,
	}

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *grpcPort))
		if err != nil {
			log.Fatal().Err(err).Msg("Couldn't listen")
		}

		master.SetHealthStatus(health.Serving)

		log.Info().Int("port", *grpcPort).Msg("gRPC serving")
		if err := server.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Couldn't listen")
		}
	}()

	log.Info().Int("port", *httpPort).Msg("HTTP serving")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("Couldn't listen")
	}
}
