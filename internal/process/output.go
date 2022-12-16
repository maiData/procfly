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
	RegisterName(string) (int, lipgloss.Style)
}

type muxWriterFactory struct {
	lck    sync.Mutex
	dst    io.Writer
	pfxlen int
	clr    map[string]lipgloss.Style
}

func NewMuxWriter(dst io.Writer) MuxWriter {
	return &muxWriterFactory{
		dst: dst,
		clr: make(map[string]lipgloss.Style),
	}
}

func (mwf *muxWriterFactory) RegisterName(name string) (int, lipgloss.Style) {
	if len(name) > mwf.pfxlen {
		mwf.pfxlen = len(name)
	}
	if _, ok := mwf.clr[name]; !ok {
		mwf.clr[name] = colors[mwf.pfxlen%len(colors)]
	}
	return mwf.pfxlen, mwf.clr[name]
}

func (mwf *muxWriterFactory) prefix(name string) []byte {
	pfxlen, style := mwf.RegisterName(name)
	templ := fmt.Sprintf("%%-%ds | ", pfxlen)
	return []byte(style.Render(fmt.Sprintf(templ, name)))
}

func (mwf *muxWriterFactory) Writer(name string) io.Writer {
	mwf.lck.Lock()
	defer mwf.lck.Unlock()
	return &muxWriter{
		muxWriterFactory: mwf,
		buf:              new(bytes.Buffer),
		name:             name,
	}
}

type muxWriter struct {
	*muxWriterFactory
	buf  *bytes.Buffer
	name string
}

func (mw *muxWriter) Write(p []byte) (int, error) {
	mw.lck.Lock()
	defer mw.lck.Unlock()

	pre := mw.prefix(mw.name)

	var beg, end int
	for i := bytes.IndexRune(p, '\n'); beg < len(p) && i >= 0; i = bytes.IndexRune(p[beg:], '\n') {
		if mw.buf.Len() == 0 {
			_, err := mw.buf.Write(pre)
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

	if end < len(p)-1 {
		_, err := mw.buf.Write(pre)
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
