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
		mwf.clr[name] = colors[len(mwf.clr)%len(colors)]
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
	mwf.RegisterName(name)
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

	rdr := bytes.NewBuffer(p)

	pre := mw.prefix(mw.name)

	count := 0
	var line []byte
	var err error
	for line, err = rdr.ReadBytes('\n'); err == nil; line, err = rdr.ReadBytes('\n') {
		if _, err := mw.dst.Write(append(pre, line...)); err != nil {
			return count, err
		}
		count += len(line)
	}

	if len(line) > 0 {
		if _, err := mw.dst.Write(append(pre, append(line, '\n')...)); err != nil {
			return count, err
		}
		count += len(line)
	}
	return count, nil
}
