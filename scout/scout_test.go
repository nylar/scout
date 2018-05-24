package scout_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/nylar/scout/discovery/discoverypb"
	"github.com/nylar/scout/files"
	"github.com/nylar/scout/files/filepb"
	"github.com/nylar/scout/health"
	"github.com/nylar/scout/scout"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// Mocks
type mockFile struct {
	name string
	dir  bool
}

func (mf *mockFile) Name() string       { return mf.name }
func (mf *mockFile) IsDir() bool        { return mf.dir }
func (mf *mockFile) Mode() os.FileMode  { panic("Not implemented") }
func (mf *mockFile) Sys() interface{}   { panic("Not implemented") }
func (mf *mockFile) Size() int64        { panic("Not implemented") }
func (mf *mockFile) ModTime() time.Time { panic("Not implemented") }

var _ os.FileInfo = &mockFile{}

type mockFilesystem struct {
	files []os.FileInfo
	err   error
}

func (mfs *mockFilesystem) ReadDir(dir string) ([]os.FileInfo, error) {
	if mfs.err != nil {
		return nil, mfs.err
	}
	return mfs.files, nil
}

var _ files.DirectoryReader = &mockFilesystem{}

type mockMasterServer struct {
	servers map[string]struct{}
	files   []*filepb.File
	err     error
}

func (mms *mockMasterServer) Sync(ctx context.Context, req *filepb.FileRequest) (*filepb.FileResponse, error) {
	if mms.err != nil {
		return nil, mms.err
	}
	mms.files = req.Files
	return &filepb.FileResponse{}, nil
}

func (mms *mockMasterServer) Register(ctx context.Context, req *discoverypb.RegisterRequest) (*discoverypb.RegisterResponse, error) {
	if mms.err != nil {
		return nil, mms.err
	}
	mms.servers[req.Directory] = struct{}{}
	return &discoverypb.RegisterResponse{}, nil
}

func (mms *mockMasterServer) Deregister(ctx context.Context, req *discoverypb.DeregisterRequest) (*discoverypb.DeregisterResponse, error) {
	if mms.err != nil {
		return nil, mms.err
	}
	delete(mms.servers, req.Directory)
	return &discoverypb.DeregisterResponse{}, nil
}

var _ filepb.FileServiceServer = &mockMasterServer{}
var _ discoverypb.DiscoveryServiceServer = &mockMasterServer{}

// Fixtures
func fileInfoFixture() []os.FileInfo {
	return []os.FileInfo{
		&mockFile{name: "foo", dir: false},
		&mockFile{name: "bar", dir: true},
		&mockFile{name: "baz", dir: false},
		&mockFile{name: "quux", dir: false},
	}
}

func filesFixture() []*filepb.File {
	return []*filepb.File{
		&filepb.File{Filename: "foo"},
		&filepb.File{Filename: "baz"},
		&filepb.File{Filename: "quux"},
	}
}

func mockServer(t *testing.T, srv *mockMasterServer) (*grpc.Server, string) {
	t.Helper()

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf(err.Error())
	}

	server := grpc.NewServer()
	filepb.RegisterFileServiceServer(server, srv)
	discoverypb.RegisterDiscoveryServiceServer(server, srv)

	go server.Serve(lis)

	return server, lis.Addr().String()
}

func TestWatcherRegister(t *testing.T) {
	dirReader := &mockFilesystem{
		files: fileInfoFixture(),
		err:   nil,
	}

	masterServer := &mockMasterServer{
		servers: make(map[string]struct{}),
	}

	server, address := mockServer(t, masterServer)
	defer server.Stop()

	rpc, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf(err.Error())
	}

	watcher, err := scout.NewWatcher(
		"/tmp",
		address,
		rpc,
		"localhost:4000",
		scout.DirectoryReader(dirReader),
		scout.SkipHiddenFiles(),
	)

	assert.NoError(t, err)

	err = watcher.Register(context.Background(), "localhost:4000")
	assert.NoError(t, err)

	assert.Equal(t, health.Serving, watcher.HealthStatus())
}

func TestWatcherDeregister(t *testing.T) {
	dirReader := &mockFilesystem{
		files: fileInfoFixture(),
		err:   nil,
	}

	masterServer := &mockMasterServer{
		servers: make(map[string]struct{}),
	}

	server, address := mockServer(t, masterServer)
	defer server.Stop()

	rpc, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf(err.Error())
	}

	watcher, err := scout.NewWatcher(
		"/tmp",
		address,
		rpc,
		"localhost:4000",
		scout.DirectoryReader(dirReader),
		scout.SkipHiddenFiles(),
	)

	assert.NoError(t, err)

	err = watcher.Register(context.Background(), "localhost:4000")
	assert.NoError(t, err)

	err = watcher.Deregister(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, health.NotServing, watcher.HealthStatus())
}

func TestWatcherSync(t *testing.T) {
	dirReader := &mockFilesystem{
		err: nil,
	}

	masterServer := &mockMasterServer{
		servers: make(map[string]struct{}),
	}

	server, address := mockServer(t, masterServer)
	defer server.Stop()

	rpc, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf(err.Error())
	}

	watcher, err := scout.NewWatcher(
		"/tmp",
		address,
		rpc,
		"localhost:4000",
		scout.DirectoryReader(dirReader),
		scout.SkipHiddenFiles(),
	)

	assert.NoError(t, err)

	err = watcher.Sync(context.Background(), filesFixture())
	assert.NoError(t, err)

	assert.Equal(t, 3, len(masterServer.files))
}

func TestWatcherClose(t *testing.T) {
	dirReader := &mockFilesystem{
		err: nil,
	}

	masterServer := &mockMasterServer{
		servers: make(map[string]struct{}),
	}

	server, address := mockServer(t, masterServer)
	defer server.Stop()

	rpc, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		t.Fatalf(err.Error())
	}

	watcher, err := scout.NewWatcher(
		"/tmp",
		address,
		rpc,
		"localhost:4000",
		scout.DirectoryReader(dirReader),
		scout.SkipHiddenFiles(),
	)

	assert.NoError(t, err)

	err = watcher.Close()
	assert.NoError(t, err)

	assert.Equal(t, 0, len(masterServer.servers))
}
