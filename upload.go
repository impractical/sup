package sup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"impractical.co/sup/magicnumber"
	"yall.in"
)

// ErrIncorrectSHA is returned when the assertion about a file's SHA doesn't
// match the contents of the file.
var ErrIncorrectSHA = errors.New("claimed SHA did not match calculated SHA")

// UploadOptions represents configuration parameters for optional behaviors of
// Upload.
type UploadOptions struct {
	// AcceptedMIMEs, if set, will only accept files with a MIME type in
	// the list.
	AcceptedMIMEs []string
}

// Upload performs a streaming upload of the data in the provided io.Reader,
// writing to the provided Storer with the provided SHA-256 hash. If the
// provided SHA-256 hash is exists in the Storer already, the File is returned,
// with the returned boolean set to false. If the hash does not exist in the
// provided Storer yet, the file is uploaded, and its File metadata returned,
// with the return boolean set to true.
//
// If the provided UploadOptions has AcceptedMIMEs set, any file uploaded will
// have its MIME type checked, and only be accepted if its MIME matches one of
// the MIME types in AcceptedMIMEs.
//
// If source is also an io.ReadCloser, its Close method will be called by
// Upload.
func Upload(ctx context.Context, s Storer, source io.Reader, sha string, opts UploadOptions) (File, bool, error) {
	log := yall.FromContext(ctx)
	log = log.WithField("sup.storer", fmt.Sprintf("%T", s))
	log = log.WithField("sup.source", fmt.Sprintf("%T", source))
	log = log.WithField("sup.claimed_sha", sha)

	// if we can, close the source when we're done
	if rc, ok := source.(io.ReadCloser); ok {
		defer rc.Close()
	}

	var writers []io.Writer
	var ctw *magicnumber.Checker

	if len(opts.AcceptedMIMEs) > 0 {
		// set up a writer that'll ensure we're only accepting files of
		// types we can support
		ctw = &magicnumber.Checker{
			SupportedMIMEs: opts.AcceptedMIMEs,
		}
		writers = append(writers, ctw)
	}

	// set up a writer that'll record the hash of the uploaded file
	hasher := sha256.New()
	writers = append(writers, hasher)

	// set up a writer that'll persist the uploaded data
	storer, err := s.Upload(yall.InContext(ctx, log), sha)
	if err != nil {
		return File{}, false, fmt.Errorf("error starting upload to %T: %s", s, err)
	}
	if storer == nil {
		log.Debug("[sup] file already exists, not re-uploading")
		file, err := s.Stat(yall.InContext(ctx, log), sha)
		if err != nil {
			return File{}, false, fmt.Errorf("error stating existing file %s: %w", sha, err)
		}
		return file, false, nil

	}
	writers = append(writers, storer)

	w := io.MultiWriter(writers...)

	log.Debug("[sup] starting upload")

	size, err := io.Copy(w, source)
	if err != nil {
		return File{}, false, fmt.Errorf("error uploading file to %T: %w", s, err)
	}

	log = log.WithField("sup.size", size)
	log.Debug("[sup] upload written")

	finalSHA := hex.EncodeToString(hasher.Sum(nil))
	log = log.WithField("sup.real_sha", finalSHA)
	if finalSHA != sha {
		log.Debug("[sup] claimed SHA did not match file, deleting")
		err = s.Delete(yall.InContext(ctx, log), sha)
		if err != nil {
			return File{}, false, fmt.Errorf("error deleting incorrectly named file %s: %w", sha, err)
		}
		log.Debug("[sup] successfully deleted")
		return File{}, false, ErrIncorrectSHA
	}
	log.Debug("[sup] completed upload")
	return File{
		SHA256:      sha,
		Size:        size,
		ContentType: ctw.MatchedMIME,
	}, true, nil
}
