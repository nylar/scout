package master

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nylar/scout/discovery/discoverypb"
	"github.com/nylar/scout/files"
	"github.com/nylar/scout/files/filepb"
	"github.com/nylar/scout/health"
	"github.com/nylar/scout/health/healthpb"
	"github.com/nylar/scout/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

const (
	clientHealthTimeout  = 200 * time.Millisecond
	healthCheckFrequency = 2 * time.Second

	unhealthyLimit = 5
)

// Master stores a list of files collected from the various connected clients
type Master struct {
	// mu protects the nodes and clients maps
	mu sync.RWMutex
	// nodes is a list of connected clients with their corresponding files
	nodes map[string][]files.File
	// clients is a list of connected clients with associated metadata
	clients map[string]*client
	// healthTicker is the frequency of health checks
	healthTicker *time.Ticker
	// healthStatus indicates the current serving status of the server
	healthStatus *int32
}

type client struct {
	directory      string
	address        string
	status         health.Status
	timesUnhealthy *int64
}

// NewMaster creates a new master server
func NewMaster() *Master {
	var status int32

	m := &Master{
		nodes:        make(map[string][]files.File),
		clients:      make(map[string]*client),
		healthTicker: time.NewTicker(healthCheckFrequency),
		healthStatus: &status,
	}
	m.SetHealthStatus(health.Unknown)

	return m
}

// Check reports the current health status back to the caller
func (m *Master) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	status := atomic.LoadInt32(m.healthStatus)

	return &healthpb.HealthCheckResponse{
		Status: health.Status(status),
	}, nil
}

// SetHealthStatus updates the health of the master server
func (m *Master) SetHealthStatus(status health.Status) {
	atomic.StoreInt32(m.healthStatus, int32(status))
}

// Sync overwrites the folder list associated with a client (directory)
func (m *Master) Sync(ctx context.Context, req *filepb.FileRequest) (*filepb.FileResponse, error) {
	m.mu.RLock()
	_, ok := m.nodes[req.Directory]
	m.mu.RUnlock()

	if !ok {
		return nil, errors.ErrNotRegistered
	}

	fileList := make([]files.File, len(req.Files))

	for i := range req.Files {
		fileList[i] = files.NewFile(req.Files[i])
	}

	m.mu.Lock()
	m.nodes[req.Directory] = fileList
	m.mu.Unlock()

	log.Info().Int("files", len(fileList)).Str("directory", req.Directory).Msg("Files synced")

	return &filepb.FileResponse{}, nil
}

// Register adds a client to the master. Each client is periodically health
// checked in case they have become orphaned.
func (m *Master) Register(ctx context.Context, req *discoverypb.RegisterRequest) (*discoverypb.RegisterResponse, error) {
	if err := m.register(req.Directory, req.Address); err != nil {
		return nil, err
	}

	return &discoverypb.RegisterResponse{}, nil
}

func (m *Master) register(directory string, address string) error {
	if directory == "" {
		return fmt.Errorf("directory can't be empty")
	}

	if address == "" {
		return fmt.Errorf("address can't be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.nodes[directory]; ok {
		return errors.ErrAlreadyRegistered
	}

	var timesUnhealthy int64

	m.nodes[directory] = nil
	m.clients[directory] = &client{
		directory:      directory,
		address:        address,
		status:         health.Serving,
		timesUnhealthy: &timesUnhealthy,
	}

	log.Info().Str("directory", directory).Str("address", address).
		Msg("Scout registered")

	return nil
}

// Deregister removes a client from the master, a client that sends subsequent
// requests will receive an error that the client is not registered.
func (m *Master) Deregister(ctx context.Context, req *discoverypb.DeregisterRequest) (*discoverypb.DeregisterResponse, error) {
	m.deregister(req.Directory)

	return &discoverypb.DeregisterResponse{}, nil
}

func (m *Master) deregister(directory string) {
	m.mu.Lock()
	delete(m.clients, directory)
	delete(m.nodes, directory)
	m.mu.Unlock()

	log.Info().Str("directory", directory).Msg("Scout deregistered")
}

// Aggregate sorts each directory into lexical order
func (m *Master) Aggregate() *files.Files {
	var size int

	m.mu.RLock()
	for _, v := range m.nodes {
		size += len(v)
	}

	files := &files.Files{
		Files: make([]files.File, 0, size),
	}

	for k, v := range m.nodes {
		cl, ok := m.clients[k]

		// If there isn't a client associated with the node, don't show its
		// files.
		if !ok {
			log.Debug().Str("directory", k).Msg("Skipping because client is missing")
			continue
		}

		// If the client is in a non-serving state, then it is either shuting
		// down or in an unknown state.
		if cl.status != health.Serving {
			log.Debug().Str("directory", k).Msg("Client isn't healthy")
			continue
		}
		files.Files = append(files.Files, v...)
	}
	m.mu.RUnlock()

	sort.Slice(files.Files, func(i, j int) bool {
		return files.Files[i].Filename < files.Files[j].Filename
	})

	return files
}

// HealthCheck periodically checks for clients that have become orphaned from
// the master server.
func (m *Master) HealthCheck() {
	for {
		select {
		case <-m.healthTicker.C:
			m.healthCheck()
		}
	}
}

func (m *Master) healthCheck() {
	m.mu.RLock()
	clients := make([]*client, 0, len(m.clients))
	for _, v := range m.clients {
		clients = append(clients, v)
	}
	m.mu.RUnlock()

	wg := &sync.WaitGroup{}
	wg.Add(len(clients))

	for _, cl := range clients {
		go func(cl *client) {
			defer wg.Done()
			if err := m.checkClientHealth(cl); err != nil {
				log.Error().Err(err).Str("directory", cl.directory).
					Msg("Unable to check node health")
				m.mu.Lock()
				cl.status = health.NotServing
				m.mu.Unlock()

				// If the current unhealthy count (plus one) is greater than the
				// maximum number of allowed unhealthy statuses in a row, we
				// should deregister the client as they have most likely been
				// abruptly killed.
				unhealthyCount := atomic.LoadInt64(cl.timesUnhealthy)
				if (unhealthyCount + 1) >= unhealthyLimit {
					m.deregister(cl.directory)
				} else {
					atomic.AddInt64(cl.timesUnhealthy, 1)
				}
			} else {
				m.mu.Lock()
				cl.status = health.Serving
				m.mu.Unlock()

				// We are healthy, reset the unhealthy counter
				atomic.StoreInt64(cl.timesUnhealthy, 0)
			}
		}(cl)
	}

	wg.Wait()
}

func (m *Master) checkClientHealth(cl *client) error {
	ctx, cancel := context.WithTimeout(context.Background(), clientHealthTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cl.address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return err
	}

	if resp.Status != health.Serving {
		return fmt.Errorf("client isn't serving")
	}

	return nil
}
