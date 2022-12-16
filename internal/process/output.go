package process

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// https://observablehq.com/@d3/color-schemes#Category10
var colors = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#1f77b4")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7f0e")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#2ca02c")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#d62728")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#9467bd")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#8c564b")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#e377c2")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#7f7f7f")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#bcbd22")),
	lipgloss.NewStyle().Foreground(lipgloss.Color("#17becf")),
}

type MuxWriter interface {
	Writer(string) io.Writer
}

type muxWriterFactory struct {
	lck sync.Mutex
	dst io.Writer
	pre string
	clr map[string]lipgloss.Style
}

func NewMuxWriter(dst io.Writer, len int) MuxWriter {
	return &muxWriterFactory{
		dst: dst,
		pre: fmt.Sprintf("%%-%ds | ", len),
		clr: make(map[string]lipgloss.Style),
	}
}

func (mwf *muxWriterFactory) Writer(name string) io.Writer {
	if _, ok := mwf.clr[name]; !ok {
		mwf.clr[name] = colors[len(mwf.clr)%len(colors)]
	}

	return &muxWriter{
		muxWriterFactory: mwf,
		buf:              new(bytes.Buffer),
		pfx:              []byte(mwf.clr[name].Render(fmt.Sprintf(mwf.pre, name))),
	}
}

type muxWriter struct {
	*muxWriterFactory
	buf *bytes.Buffer
	pfx []byte
}

func (mw *muxWriter) Write(p []byte) (int, error) {
	mw.lck.Lock()
	defer mw.lck.Unlock()

	var beg, end int
	for i := bytes.IndexRune(p, '\n'); beg < len(p) && i >= 0; i = bytes.IndexRune(p[beg:], '\n') {
		if mw.buf.Len() == 0 {
			_, err := mw.buf.Write(mw.pfx)
			if err != nil {
				return beg, err
			}
		}

		end = beg + i
		_, err := mw.buf.Write(p[beg : end+1])
		if err != nil {
			return beg, err
		}
		beg = end + 1

		_, err = mw.buf.WriteTo(mw.dst)
		if err != nil {
			return end, err
		}
	}

	if end < len(p) {
		_, err := mw.buf.Write(mw.pfx)
		if err != nil {
			return beg, err
		}

		_, err = mw.buf.Write(p[end+1:])
		if err != nil {
			return end, err
		}
	}

	return len(p), nil
}
