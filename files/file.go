package files

import (
	"os"

	"github.com/nylar/scout/files/filepb"
)

// Compile time check to ensure DirReader implements DirectoryReader
var _ DirectoryReader = DirReader{}

// Files is a list of File objects
type Files struct {
	Files []File `json:"files"`
}

// File represents a single file
type File struct {
	Filename string `json:"filename"`
}

// NewFile converts a transport file object into a presentation file object.
func NewFile(f *filepb.File) File {
	return File{
		Filename: f.Filename,
	}
}

// DirectoryReader describes a way of reading a directory
type DirectoryReader interface {
	ReadDir(string) ([]os.FileInfo, error)
}

// DirReader implements the DirectoryReader interface
type DirReader struct{}

// ReadDir reads each file from a directory, it is similar to io/ioutil.ReadDir
// except it doesn't sort the file list.
func (DirReader) ReadDir(dir string) ([]os.FileInfo, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}

	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	return list, nil
}
