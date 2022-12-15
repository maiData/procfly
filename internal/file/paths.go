package file

import "path/filepath"

type Paths struct {
	RootDir     string
	ProcflyFile string
}

func (p Paths) Rel(file ...string) string {
	return filepath.Join(append([]string{p.RootDir}, file...)...)
}

func NewPaths(root string) Paths {
	return Paths{
		RootDir:     root,
		ProcflyFile: filepath.Join(root, "procfly.yml"),
	}
}
