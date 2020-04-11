package sup

import (
	"context"
	"fmt"
	"io"

	"yall.in"
)

// Download writes the file at the provided path inside the provided Storer to
// the provided io.Writer, returning an ErrFileNotFound error if the path does
// not exist inside the Storer.
//
// If the provided io.Writer is also an io.WriteCloser, its Close method will
// be called by Download.
func Download(ctx context.Context, s Storer, dst io.Writer, sha string) error {
	log := yall.FromContext(ctx)
	log = log.WithField("sup.storer", fmt.Sprintf("%T", s))
	log = log.WithField("sup.destination", fmt.Sprintf("%T", dst))
	log = log.WithField("sup.sha", sha)

	log.Debug("[sup] downloading")

	// if our destination can be closed, close it when we're done
	if wc, ok := dst.(io.WriteCloser); ok {
		defer wc.Close()
	}

	// get a reader from our Storer
	rc, err := s.Download(yall.InContext(ctx, log), sha)
	if err != nil {
		return fmt.Errorf("error starting download from %T: %w", s, err)
	}
	defer rc.Close()

	log.Debug("[sup] starting data copy")
	_, err = io.Copy(dst, rc)
	if err != nil {
		return fmt.Errorf("error copying information from %T to %T: %w", dst, rc, err)
	}

	log.Debug("[sup] download complete")
	return nil
}
