package memory

import (
	"bytes"
	"context"
	"io"

	"github.com/h2non/filetype"
	memdb "github.com/hashicorp/go-memdb"
	"impractical.co/sup"
)

var _ sup.Storer = &Storer{}

var (
	schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"file": &memdb.TableSchema{
				Name: "file",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID", Lowercase: true},
					},
				},
			},
		},
	}
)

type File struct {
	ID          string
	Size        int64
	ContentType string
	Contents    []byte
	buf         *bytes.Buffer
}

func NewFile(id string, in []byte) *File {
	b := bytes.NewBuffer(in)
	return &File{
		ID:       id,
		Contents: in,
		buf:      b,
	}
}

func (f *File) Write(p []byte) (int, error) {
	n, err := f.buf.Write(p)
	if err != nil {
		return n, err
	}
	f.Contents = append(f.Contents, p...)
	return n, err
}

func (f *File) Read(p []byte) (int, error) {
	return f.buf.Read(p)
}

func (f *File) Close() error {
	f.buf = bytes.NewBuffer(f.Contents)
	return nil
}

type Storer struct {
	db *memdb.MemDB
}

func (s *Storer) Upload(ctx context.Context, hash string) (io.WriteCloser, error) {
	txn := s.db.Txn(true)
	defer txn.Abort()
	exists, err := txn.First("file", "id", hash)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, nil
	}
	f := NewFile(hash, []byte{})
	err = txn.Insert("file", f)
	if err != nil {
		return nil, err
	}
	txn.Commit()
	return f, nil
}

func (s *Storer) Download(ctx context.Context, hash string) (io.ReadCloser, error) {
	txn := s.db.Txn(false)
	res, err := txn.First("file", "id", hash)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, sup.ErrFileNotFound
	}
	return res.(*File), nil
}

func (s *Storer) Delete(ctx context.Context, hash string) error {
	txn := s.db.Txn(true)
	defer txn.Abort()
	exists, err := txn.First("file", "id", hash)
	if err != nil {
		return err
	}
	if exists == nil {
		return nil
	}
	err = txn.Delete("file", exists)
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (s *Storer) Stat(ctx context.Context, hash string) (sup.File, error) {
	txn := s.db.Txn(false)
	res, err := txn.First("file", "id", hash)
	if err != nil {
		return sup.File{}, err
	}
	if res == nil {
		return sup.File{}, sup.ErrFileNotFound
	}
	f := res.(*File)
	t, err := filetype.Match(f.Contents)
	if err != nil {
		return sup.File{}, err
	}
	return sup.File{
		ContentType: t.MIME.Value,
		Size:        int64(len(f.Contents)),
		SHA256:      hash,
	}, nil
}

func NewStorer() (*Storer, error) {
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}
	return &Storer{
		db: db,
	}, nil
}
