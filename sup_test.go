package sup_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"impractical.co/sup"
	"impractical.co/sup/memory"
	yall "yall.in"
	"yall.in/colour"
)

type Factory interface {
	NewStorer(ctx context.Context) (sup.Storer, error)
	TeardownStorers() error
}

var factories []Factory

func TestMain(m *testing.M) {
	flag.Parse()

	// set up our test storers
	factories = append(factories, memory.Factory{})

	// run the tests
	result := m.Run()

	// tear down all the storers we created
	for _, factory := range factories {
		err := factory.TeardownStorers()
		if err != nil {
			log.Printf("Error cleaning up after %T: %+v\n", factory, err)
		}
	}

	// return the test result
	os.Exit(result)
}

func runTest(t *testing.T, f func(*testing.T, sup.Storer, context.Context)) {
	t.Parallel()
	logger := yall.New(colour.New(os.Stdout, yall.Debug))
	for _, factory := range factories {
		ctx := yall.InContext(context.Background(), logger)
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %+v\n", factory, err)
		}
		t.Run(fmt.Sprintf("Storer=%T", storer), func(t *testing.T) {
			t.Parallel()
			f(t, storer, ctx)
		})
	}
}

func TestUploadDownloadDelete(t *testing.T) {
	type input struct {
		hash string
		data []byte
	}
	type output struct {
		file sup.File
		err  error
	}
	type uploadTest struct {
		in  input
		out output
	}
	table := map[string]uploadTest{
		"helloworld": uploadTest{
			in:  input{data: []byte("hello, world"), hash: "09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b"},
			out: output{file: sup.File{SHA256: "09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b", Size: 12}},
		},
		"gif": uploadTest{
			in:  input{data: testgif, hash: "1c86d1005b0a4114c013c019653026bb955ada19344ca4391a2b6f72febf8668"},
			out: output{file: sup.File{SHA256: "1c86d1005b0a4114c013c019653026bb955ada19344ca4391a2b6f72febf8668", Size: 1377666, ContentType: "image/gif"}},
		},
	}
	var expectedMIMEs []string
	for _, row := range table {
		if row.out.file.ContentType != "" {
			expectedMIMEs = append(expectedMIMEs, row.out.file.ContentType)
		}
	}
	for id, testcase := range table {
		id, testcase := id, testcase
		t.Run("ID="+id, func(t *testing.T) {
			runTest(t, func(t *testing.T, storer sup.Storer, ctx context.Context) {
				buffer := bytes.NewBuffer(testcase.in.data)
				result, created, err := sup.Upload(ctx, storer, ioutil.NopCloser(buffer), testcase.in.hash, sup.UploadOptions{AcceptedMIMEs: expectedMIMEs})
				if !errors.Is(err, testcase.out.err) {
					t.Errorf("Expected error to be %q, got %q", testcase.out.err, err)
					return
				}
				if !created {
					t.Errorf("Expected new file to be created, was not")
					return
				}

				if result.Size != testcase.out.file.Size {
					t.Errorf("Expected size to be %d, got %d", testcase.out.file.Size, result.Size)
					return
				}
				if result.SHA256 != testcase.out.file.SHA256 {
					t.Errorf("Expected SHA256 to be %q, got %q", testcase.out.file.SHA256, result.SHA256)
					return
				}
				if result.ContentType != testcase.out.file.ContentType {
					t.Errorf("Expected content type to be %q, got %q", testcase.out.file.ContentType, result.ContentType)
					return
				}

				ctx = context.Background()
				var buf bytes.Buffer
				err = sup.Download(ctx, storer, &buf, testcase.out.file.SHA256)
				if !errors.Is(err, testcase.out.err) {
					t.Errorf("Expected error to be %q, got %q", testcase.out.err, err)
					return
				}
				b := buf.Bytes()
				if !bytes.Equal(testcase.in.data, b) {
					t.Errorf("Expected download to be %q, got %q", hex.EncodeToString(testcase.in.data), hex.EncodeToString(b))
					return
				}

				ctx = context.Background()
				err = storer.Delete(ctx, testcase.out.file.SHA256)
				if err != nil {
					t.Errorf("Unexpected error: %s", err)
					return
				}
				buf = bytes.Buffer{}
				err = sup.Download(ctx, storer, &buf, testcase.out.file.SHA256)
				if !errors.Is(err, sup.ErrFileNotFound) {
					t.Errorf("Expected %q, got %q", sup.ErrFileNotFound, err)
					return
				}
				err = storer.Delete(ctx, testcase.out.file.SHA256)
				if err != nil {
					t.Errorf("Unexpected error: %s", err)
					return
				}
			})
		})
	}
}

// TODO(paddy): test a file that lies about its hash

// TODO(paddy): test uploading a file with the same hash as an existing file

// TODO(paddy): test uploading a file claiming its hash is the same as an existing file's when it isn't
// we need to be careful that doesn't delete the legitimate file!
