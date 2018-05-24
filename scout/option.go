package scout

import (
	"github.com/nylar/scout/files"
)

// Option defines additional configuration for a Watcher
type Option func(*Watcher) error

// SkipHiddenFiles ensures that all files that begin with a period are skipped
// when a directory is read.
func SkipHiddenFiles() Option {
	return func(w *Watcher) error {
		return w.setSkipHiddenFiles(true)
	}
}

// DirectoryReader allows you to supply your own directory reader
func DirectoryReader(dr files.DirectoryReader) Option {
	return func(w *Watcher) error {
		return w.setDirectoryReader(dr)
	}
}
