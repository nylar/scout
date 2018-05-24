package scout

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nylar/scout/discovery/discoverypb"
	"github.com/nylar/scout/files"
	"github.com/nylar/scout/files/filepb"
	"github.com/nylar/scout/health"
	"github.com/nylar/scout/health/healthpb"
	"github.com/nylar/scout/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	defaultSyncDuration        = 1 * time.Second
	defaultHealthCheckDuration = 100 * time.Millisecond

	defaultReregisterDuration = 250 * time.Millisecond

	defaultBurstRate = 50
	defaultLimit     = rate.Limit(10)

	defaultHealthCheckFrequency = 1 * time.Second
	defaultPollFrequency        = 5 * time.Second

	defaultDeregisterDuration = 1 * time.Second
)

// Watcher monitors files in a directory and reports changes to a master
// server.
type Watcher struct {
	// mu protects the files slice
	mu sync.RWMutex
	// files is the current list of files in the directory that is being
	// watched.
	files []*filepb.File
	// masterServerAddress is the address of the scout master
	masterServerAddress string
	// clientAddress is the address of the scout
	clientAddress string
	// skipHiddenFiles indicates whether we want to ignore files that begin
	// with a period.
	skipHiddenFiles bool
	// directory is the folder that is being monitored.
	directory string
	// watcher monitors the directory for any changes and notifies the
	// watcher to scan the directory
	watcher *fsnotify.Watcher
	// notifier responds to changes in the directory
	notifier chan struct{}
	// stop signals each background task to terminate.
	stop chan struct{}
	// rateLimiter prevents the watcher from being flooded with filesystem
	// changes.
	rateLimiter *rate.Limiter
	// fileTicker is the period between each poll of the filesystem.
	fileTicker *time.Ticker
	// fileServiceClient handles the synchronisation between the client and
	// the server.
	fileServiceClient filepb.FileServiceClient
	// discoveryServiceClient handles service discovery between the client
	// and the server.
	discoveryServiceClient discoverypb.DiscoveryServiceClient
	// directoryReader reads a list of files from a directory
	directoryReader files.DirectoryReader
	// healthStatus indicates the current serving status of the client
	healthStatus *int32
	// healthClient is used to periodically check if the master server is alive
	healthClient healthpb.HealthClient
	// healthTicket is the period between each health check
	healthTicker *time.Ticker
}

// NewWatcher creates a directory watcher.
func NewWatcher(directory, masterAddress string, conn *grpc.ClientConn, clientAddress string, opts ...Option) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(directory); err != nil {
		return nil, err
	}

	var status int32

	w := &Watcher{
		masterServerAddress:    masterAddress,
		clientAddress:          clientAddress,
		directory:              directory,
		watcher:                watcher,
		notifier:               make(chan struct{}),
		stop:                   make(chan struct{}),
		rateLimiter:            rate.NewLimiter(defaultLimit, defaultBurstRate),
		fileTicker:             time.NewTicker(defaultPollFrequency),
		fileServiceClient:      filepb.NewFileServiceClient(conn),
		discoveryServiceClient: discoverypb.NewDiscoveryServiceClient(conn),
		directoryReader:        files.DirReader{},
		healthStatus:           &status,
		healthClient:           healthpb.NewHealthClient(conn),
		healthTicker:           time.NewTicker(defaultHealthCheckFrequency),
	}
	w.setHealthStatus(health.Unknown)

	for _, opt := range opts {
		if err := opt(w); err != nil {
			return nil, err
		}
	}

	return w, nil
}

func (w *Watcher) setHealthStatus(status health.Status) {
	atomic.StoreInt32(w.healthStatus, int32(status))
}

// HealthStatus retrieves the current status of the scout
func (w *Watcher) HealthStatus() health.Status {
	return health.Status(atomic.LoadInt32(w.healthStatus))
}

// Close halts the background tasks, deregisters the client with the master
// server and stops watching the filesystem.
func (w *Watcher) Close() error {
	w.fileTicker.Stop()
	w.healthTicker.Stop()
	close(w.stop)

	ctx, cancel := context.WithTimeout(context.Background(), defaultDeregisterDuration)
	defer cancel()

	if err := w.Deregister(ctx); err != nil {
		return err
	}

	return w.watcher.Close()
}

// Sync sends a list of unsorted files to the master server.
func (w *Watcher) Sync(ctx context.Context, files []*filepb.File) error {
	req := &filepb.FileRequest{
		Directory: w.directory,
		Files:     files,
	}

	_, err := w.fileServiceClient.Sync(ctx, req)
	return err
}

// Register associates the current client with the master server.
func (w *Watcher) Register(ctx context.Context, address string) error {
	_, err := w.discoveryServiceClient.Register(ctx, &discoverypb.RegisterRequest{
		Directory: w.directory,
		Address:   address,
	})
	if err != nil {
		return err
	}
	w.setHealthStatus(health.Serving)

	return nil
}

// Deregister dissociates the current client with the master server.
func (w *Watcher) Deregister(ctx context.Context) error {
	w.setHealthStatus(health.NotServing)
	_, err := w.discoveryServiceClient.Deregister(ctx, &discoverypb.DeregisterRequest{
		Directory: w.directory,
	})
	return err
}

// Check reports the current health status back to the caller
func (w *Watcher) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{
		Status: w.HealthStatus(),
	}, nil
}

// Start initialises the notification handler and begins handling filesystem
// events.
func (w *Watcher) Start() {
	go w.notificationHandler()
	go w.healthCheck()

	for {
		select {
		case event := <-w.watcher.Events:
			if w.rateLimiter.Allow() {
				// NOTE: If we only care about the list of files
				// in a directory, then we probably don't need
				// to care about files that are being modified.
				if event.Op&fsnotify.Write == fsnotify.Write {
					w.notifier <- struct{}{}
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					w.notifier <- struct{}{}
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					w.notifier <- struct{}{}
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					w.notifier <- struct{}{}
				}
			}
		case err := <-w.watcher.Errors:
			log.Error().Msg(err.Error())
		case <-w.stop:
			return
		}
	}
}

func (w *Watcher) healthCheck() {
	for {
		select {
		case <-w.healthTicker.C:
			oldStatus := w.HealthStatus()

			status, err := w.sendHealthCheck()
			if err != nil {
				if status != oldStatus {
					log.Warn().Str("before", w.HealthStatus().String()).
						Str("after", status.String()).Msg("Server health changed")
					w.setHealthStatus(status)
				}
			} else {
				// The master server was previously unavailable but has begun
				// serving again. We should re-register the client and perform
				// a file sync.
				if (oldStatus == health.NotServing || oldStatus == health.Unknown) && status == health.Serving {
					ctx, cancel := context.WithTimeout(context.Background(), defaultReregisterDuration)
					func() {
						defer cancel()
						if err := w.Register(ctx, w.clientAddress); err != nil {
							log.Error().Err(err).Msg("Couldn't re-register the client")
							return // Still unhealthy
						}

						// Trigger a file update
						w.notifier <- struct{}{}

						w.setHealthStatus(status)

						log.Info().Str("before", oldStatus.String()).
							Str("after", status.String()).Msg("Server health changed")
					}()
				} else {
					w.setHealthStatus(status)
				}
			}
		case <-w.stop:
			return
		}
	}
}

func (w *Watcher) sendHealthCheck() (health.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHealthCheckDuration)
	defer cancel()

	resp, err := w.healthClient.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return health.Unknown, err
	}

	if resp.Status != health.Serving {
		return resp.Status, fmt.Errorf("master isn't serving")
	}

	return resp.Status, nil
}

func (w *Watcher) notificationHandler() {
	files, err := w.readDirectory()
	if err != nil {
		log.Error().Err(err).Msg("Unable to read directory")
	} else {
		w.send(files, true)
	}

	for {
		select {
		case <-w.notifier:
			files, err := w.readDirectory()
			if err != nil {
				log.Error().Err(err).Msg("Unable to read directory")
				continue
			}

			w.send(files, true)
		case <-w.fileTicker.C:
			files, err := w.readDirectory()
			if err != nil {
				log.Error().Err(err).Msg("Unable to read directory")
				continue
			}

			w.send(files, false)
		case <-w.stop:
			return
		}
	}
}

// readDirectory attempts to read the name of each file in a directory
func (w *Watcher) readDirectory() ([]*filepb.File, error) {
	fileList, err := w.directoryReader.ReadDir(w.directory)
	if err != nil {
		return nil, err
	}

	fileNames := make([]*filepb.File, len(fileList))
	var j int

	for i := range fileList {
		// Ignore directories
		if fileList[i].IsDir() {
			continue
		}

		// Skip files that begin with a period if configured to do so
		if w.skipHiddenFiles {
			if strings.HasPrefix(fileList[i].Name(), ".") {
				continue
			}
		}
		fileNames[j] = &filepb.File{
			Filename: fileList[i].Name(),
		}
		j++
	}

	// Adjust the slice to account for any files that were filtered
	fileNames = fileNames[:j]

	return fileNames, nil
}

func (w *Watcher) send(files []*filepb.File, force bool) {
	shouldSend := true

	// A signal from the filesystem that a change has occurred or if the
	// the scoutmaster has returned to a serving state should trigger a file
	// sync. Regular polling is therefore subject to a equality check, if the
	// files haven't changed, we don't need to inform the scoutmaster
	if !force {
		w.mu.RLock()
		shouldSend = !fileSliceEqual(w.files, files)
		w.mu.RUnlock()
	}

	// Only send if the master is serving and the clients file list is
	// different from the master's.
	if w.HealthStatus() == health.Serving && shouldSend {
		w.mu.Lock()
		w.files = files
		w.mu.Unlock()

		go func(files []*filepb.File) {
			ctx, cancel := context.WithTimeout(context.Background(), defaultSyncDuration)
			defer cancel()
			if err := w.Sync(ctx, files); err != nil {
				s, _ := status.FromError(err)

				switch {
				case s.Message() == errors.ErrNotRegistered.Error():
					if err := w.Register(ctx, w.clientAddress); err != nil {
						log.Error().Err(err).Msg("Couldn't re-register client")
					}
				default:
					log.Error().Err(err).Msg("Couldn't sync files with master")
				}
			}
		}(files)
	}
}

func (w *Watcher) setSkipHiddenFiles(skip bool) error {
	w.skipHiddenFiles = skip
	return nil
}

func (w *Watcher) setDirectoryReader(dr files.DirectoryReader) error {
	w.directoryReader = dr
	return nil
}

// fileSliceEqual determines if two slices of files are the same.
func fileSliceEqual(a, b []*filepb.File) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if *a[i] != *b[i] {
			return false
		}
	}

	return true
}
