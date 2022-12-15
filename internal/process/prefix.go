package process

import (
	"bytes"
	"io"
	"sync"
)

type PrefixWriterFactory interface {
	Writer(string) io.Writer
}

type writerFunc func(p []byte) (int, error)

func (wf writerFunc) Write(p []byte) (int, error) {
	return wf(p)
}

func NewPrefixWriterFactory(dst io.Writer) PrefixWriterFactory {
	return &prefixWriter{dst: dst}
}

type prefixWriter struct {
	lck sync.Mutex
	dst io.Writer
}

func (pw *prefixWriter) Writer(prefix string) io.Writer {
	buf := new(bytes.Buffer)
	return writerFunc(func(p []byte) (int, error) {
		pw.lck.Lock()
		defer pw.lck.Unlock()

		var beg, end int
		for i := bytes.IndexRune(p, '\n'); beg < len(p) && i >= 0; i = bytes.IndexRune(p[beg:], '\n') {
			if buf.Len() == 0 {
				_, err := buf.Write([]byte(prefix))
				if err != nil {
					return beg, err
				}
			}

			end = beg + i
			_, err := buf.Write(p[beg : end+1])
			if err != nil {
				return beg, err
			}
			beg = end + 1

			_, err = buf.WriteTo(pw.dst)
			if err != nil {
				return end, err
			}
		}

		if end < len(p) {
			_, err := buf.Write([]byte(prefix))
			if err != nil {
				return beg, err
			}

			_, err = buf.Write(p[end+1:])
			if err != nil {
				return end, err
			}
		}

		return len(p), nil
	})
}
