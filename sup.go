package sup

import (
	"context"
	"errors"
	"io"
)

// ErrFileNotFound is returned when a File is requested and can't be found.
var ErrFileNotFound = errors.New("file not found")

// File represents an uploaded file.
type File struct {
	SHA256      string
	Size        int64
	ContentType string
}

// Storer represents a destination for uploaded files.
type Storer interface {
	Upload(ctx context.Context, sha string) (io.WriteCloser, error)
	Download(ctx context.Context, sha string) (io.ReadCloser, error)
	Delete(ctx context.Context, sha string) error
	Stat(ctx context.Context, sha string) (File, error)
}
