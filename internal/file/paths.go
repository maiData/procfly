package file

import (
	"os"
	"path/filepath"
	"strings"
)

type Paths struct {
	RootDir     string
	ProcflyFile string
}

func (p Paths) Open(file string, flag int, perm os.FileMode) (*os.File, error) {
	path := p.normalize(file)

	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return nil, err
	}

	return os.OpenFile(path, flag, perm)
}

func (p Paths) Read(path string) ([]byte, error) {
	return os.ReadFile(p.normalize(path))
}

func (p Paths) normalize(file string) string {
	if strings.HasPrefix(file, "/") {
		return file
	}
	return filepath.Join(p.RootDir, file)
}

func NewPaths(root string) Paths {
	return Paths{
		RootDir:     root,
		ProcflyFile: filepath.Join(root, "procfly.yml"),
	}
}
