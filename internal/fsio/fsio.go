package fsio

import "os"

const DefaultBufferSize = 512

type Reader interface {
	Open(name string) (*os.File, error)
	ReadFile(name string) ([]byte, error)
	ReadBufferFromFile(file *os.File) ([]byte, error)
}

type Writer interface {
	Create(name string) (*os.File, error)
	Write(file *os.File, buf []byte) error
}

type RealReader struct {
	BufferSize int
}

func NewRealReader(bufferSize int) *RealReader {
	return &RealReader{BufferSize: bufferSize}
}

func (r *RealReader) Open(name string) (*os.File, error) { return os.Open(name) }

func (r *RealReader) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

func (r *RealReader) ReadBufferFromFile(file *os.File) ([]byte, error) {
	buf := make([]byte, r.BufferSize)
	_, err := file.Read(buf)
	return buf, err
}

type RealWriter struct{}

func (w *RealWriter) Create(name string) (*os.File, error) { return os.Create(name) }

func (w *RealWriter) Write(file *os.File, buf []byte) error {
	_, err := file.Write(buf)
	return err
}
