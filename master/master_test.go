package master_test

import (
	"context"
	"sort"
	"testing"

	"github.com/nylar/scout/discovery/discoverypb"
	"github.com/nylar/scout/files/filepb"

	"github.com/nylar/scout/master"
	"github.com/stretchr/testify/assert"
)

// Fixtures
func clientFiles() map[string][]*filepb.File {
	return map[string][]*filepb.File{
		"foo": {
			&filepb.File{Filename: "3"},
			&filepb.File{Filename: "2"},
		},
		"bar": {
			&filepb.File{Filename: "4"},
			&filepb.File{Filename: "1"},
		},
	}
}

func TestMasterAggregate(t *testing.T) {
	scoutMaster := master.NewMaster()

	for dir, fileList := range clientFiles() {
		if _, err := scoutMaster.Register(context.Background(), &discoverypb.RegisterRequest{
			Directory: dir,
			Address:   "127.0.0.1:0",
		}); err != nil {
			t.Fatalf(err.Error())
		}
		defer scoutMaster.Deregister(context.Background(), &discoverypb.DeregisterRequest{
			Directory: dir,
		})

		if _, err := scoutMaster.Sync(context.Background(), &filepb.FileRequest{
			Directory: dir,
			Files:     fileList,
		}); err != nil {
			t.Fatalf(err.Error())
		}
	}

	sortedFiles := scoutMaster.Aggregate()

	assert.Equal(t, 4, len(sortedFiles.Files))

	assert.True(t, sort.SliceIsSorted(sortedFiles.Files, func(i, j int) bool {
		return sortedFiles.Files[i].Filename < sortedFiles.Files[j].Filename
	}))
}
