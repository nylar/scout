package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/nylar/scout/health/healthpb"

	"github.com/nylar/scout/scout"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

const directoryKey = "MONITOR_DIR"
const scoutMasterAddressKey = "SCOUT_MASTER_ADDRESS"

var errPathIsRelative = errors.New("scout: directory path must be absolute")

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func validDirectory(dir string) error {
	if !filepath.IsAbs(dir) {
		return errPathIsRelative
	}

	fileInfo, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return fmt.Errorf("scout: %s is not a directory", dir)
	}

	return nil
}

func main() {
	directory := filepath.Clean(os.Getenv(directoryKey))
	scoutMasterAddress := os.Getenv(scoutMasterAddressKey)

	if err := validDirectory(directory); err != nil {
		log.Fatal().Err(err).Str("directory", directory).
			Msg("Not a valid directory")
	}

	rpc, err := grpc.Dial(scoutMasterAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Str("address", scoutMasterAddress).
			Msg("Couldn't connect to server")
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to create listener")
	}

	address := listener.Addr().String()

	watcher, err := scout.NewWatcher(directory, scoutMasterAddress, rpc, address, scout.SkipHiddenFiles())
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't create file watcher")
	}

	log.Info().Str("directory", directory).
		Msg("Started watching directory")

	if err := watcher.Register(context.Background(), address); err != nil {
		log.Fatal().Err(err).Msg("Couldn't register scout")
	}

	go watcher.Start()

	server := grpc.NewServer()
	healthpb.RegisterHealthServer(server, watcher)

	signals := []os.Signal{os.Interrupt, syscall.SIGINT}

	signalChan := make(chan os.Signal, len(signals))
	done := make(chan struct{})

	signal.Notify(signalChan, signals...)
	go func(funcs ...io.Closer) {
		<-signalChan
		log.Info().Msg("Shutting down")

		for _, f := range funcs {
			if err := f.Close(); err != nil {
				log.Error().Err(err).Msg("Closing failure")
			}
		}

		server.GracefulStop()

		log.Info().Msg("Finished")

		close(done)
	}(watcher)

	log.Info().Str("address", address).Msg("Started server")
	if err := server.Serve(listener); err != nil {
		log.Fatal().Err(err).Str("address", address).Msg("Couldn't start server")
	}

	<-done
}
