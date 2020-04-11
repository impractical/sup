package magicnumber

import "testing"

type testCase struct {
	in          [][]byte
	writeErr    []error
	closeErr    error
	MatchedMIME string
}

func padBytes(in []byte) []byte {
	for len(in) < 300 {
		in = append(in, in...)
	}
	return in
}

var testCases = map[string]testCase{
	"FileTooShort": {
		in:       [][]byte{testgif[:259]},
		closeErr: ErrUnsupportedFile,
	},
	"FileTypeNotSupported": {
		in:       [][]byte{padBytes([]byte("this is a text/plain file, which is a real MIME type but isn't one of the supported mimes."))},
		writeErr: []error{ErrUnsupportedFile},
		closeErr: ErrUnsupportedFile,
	},
	"GIF": {
		in:          [][]byte{testgif},
		MatchedMIME: "image/gif",
	},
	"GIFMultiWrites": {
		in:          [][]byte{testgif[:100], testgif[100:512], testgif[512:]},
		MatchedMIME: "image/gif",
	},
}

func TestChecker(t *testing.T) {
	t.Parallel()
	for l, c := range testCases {
		label := l
		testCase := c
		t.Run(label, func(t *testing.T) {
			t.Parallel()
			checker := &Checker{
				SupportedMIMEs: []string{
					"image/gif", "image/jpeg", "image/png",
				},
			}
			for pos, in := range testCase.in {
				_, err := checker.Write(in)
				var expected error
				if len(testCase.writeErr) > pos {
					expected = testCase.writeErr[pos]
				}
				if err != expected {
					t.Errorf("Expected error on write to be %q, got %q with detected MIME type %q", expected, err, checker.MatchedMIME)
					return
				}
			}
			err := checker.Close()
			if err != testCase.closeErr {
				t.Errorf("Expected error on close to be %q, got %q with detected MIME type %q", testCase.closeErr, err, checker.MatchedMIME)
				return
			}
			if checker.MatchedMIME != testCase.MatchedMIME {
				t.Errorf("Expected matched MIME to be %q, got %q", testCase.MatchedMIME, checker.MatchedMIME)
			}
		})
	}
}
