package magicnumber

import (
	"errors"

	"github.com/h2non/filetype"
)

// the minimum number of bytes needed to determine the MIME type.
const minBytesNeeded = 261

// ErrUnsupportedFile is returned when the detected MIME type of the file isn't
// in the SupportedMIMEs list or the file is not large enough for us to detect
// a MIME type on it.
var ErrUnsupportedFile = errors.New("unsupported file")

// Checker is an io.WriteCloser that will check to see if the data passed to it
// is for a file with a MIME type in SupportedMIMEs. If so, MatchedMIME will be
// set to the MIME type matched and Write and Close will return no error.
// Otherwise, ErrUnsupportedFile is returned, either from Write as soon as we
// can tell what MIME type the file is, or from Close if no MIME type has been
// detected.
type Checker struct {
	buf            []byte
	SupportedMIMEs []string
	MatchedMIME    string
	passed         bool
}

// Write checks the incoming data for magic number bytes that will indicate the
// MIME type of the data. Once a MIME type is matched, no more data is read
// into memory, and the function is a no-op. If a MIME type is detected that
// isn't in SupportedMIMEs, ErrUnsupportedFile is returned.
func (m *Checker) Write(b []byte) (int, error) {
	if m.MatchedMIME != "" {
		return len(b), nil
	}
	m.buf = append(m.buf, b...)
	if len(m.buf) < minBytesNeeded {
		return len(b), nil
	}
	for _, mime := range m.SupportedMIMEs {
		if filetype.IsMIME(m.buf, mime) {
			m.MatchedMIME = mime
			m.buf = nil
			return len(b), nil
		}
	}
	return len(b), ErrUnsupportedFile
}

// Close returns ErrUnsupportedFile if no MIME type was detected; this usually
// indicates a file that is too small for the Checker to detect a MIME type
// automatically. If the Checker identified a supported MIME type, no error is
// returned.
func (m *Checker) Close() error {
	if m.MatchedMIME != "" {
		return nil
	}
	return ErrUnsupportedFile
}
