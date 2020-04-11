# sup

`sup` is a package providing helpers for streaming upload and download of
content-addressed media. It is meant to be pluggable for storage interfaces,
and optionally allows limiting the filetypes that can be uploaded. It aims to
keep the minimal amount of data in memory or on disk at any given point, and to
abort as early as possible.

## Usage

The main usage of `sup` is through the `Upload` function. It takes a
destination to write the data to, a source to read the data from, an assertion
about what the final SHA-256 hash of the data will be, and a struct of options
that control optional upload behavior. It returns a `File` struct, a boolean
indicating whether the file was a new upload (`true` indicates the file didn't
exist in the destination yet and was uploaded fully, `false` indicates the file
already exists in the destination and wasn't re-uploaded),  and an error.

A mirror `Download` function is provided, that streams a previously uploaded
file from the upload destination to an `io.Writer`.

`sup` will always attempt to close any source or destination that fills the
`io.Closer` interface.

### Storers

Storers are the upload destinations that fill the `Storer` interface. Their
methods are described below.

#### Upload

`Storer` implementations must fill the `Upload` method, which accepts an
`io.WriteCloser` and a path. The `Storer` should persist the data written to
the `io.WriteCloser` in such a way that it can be retrieved by specifying the
same path. The data should _not_ be persisted until `Close` is called on the
`WriteCloser`.

They must provide an Upload method that will return an `io.WriteCloser` that
the provided path can be written to; this is where the data being uploaded will
be written.  They must also provide a Delete method that will remove a file
that has been uploaded, and a Stat method that will retrieve the information
needed for a `File` struct.

#### Download

`Storer` implementations must fill the `Download` method, which accepts a path
and returns an `io.ReadCloser`. The returned `io.ReadCloser` should return an
`io.EOF` when there's no more data to be read.

#### Delete

`Storer` implementations must fill the `Delete` method, which accepts a path
and returns an error. Once `Delete` is called, calls to `Download` and `Stat`
should not be able to find the file anymore, and `Upload` should not see the
file as uploaded anymore when doing its duplication checks.

#### Stat

`Storer` implementations must fill the `Stat` method, which accepts a path and
returns the information necessary to build a File struct, if the path exists.

### UploadOptions

The `UploadOptions` struct contains the configuration parameters to control the
optional behavior of the `Upload` function. Each option has more information
contained below.

#### AcceptedMIMEs

`AcceptedMIMEs`, when set, will examine the file being uploaded as it is
uploaded and check to make sure the detected MIME type of the file is one of
the listed types. If not, an error will be returned and the upload will be
aborted.

### File structs

The `File` type represents an uploaded file and contains metadata about it. It
contains the SHA256 hash of the file, the size of the file in bytes, and the
detected MIME type of the file's contents.
