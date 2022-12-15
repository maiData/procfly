package process_test

import (
	"os"

	"github.com/emm035/procfly/internal/process"
)

func ExamplePrefixWriterFactory() {
	pwf := process.NewPrefixWriterFactory(os.Stdout)
	prw := pwf.Writer("a")

	_, err := prw.Write([]byte("bc\ndef"))
	if err != nil {
		return
	}

	_, err = prw.Write([]byte("ghi\n"))
	if err != nil {
		return
	}

	_, err = prw.Write([]byte("jkl\nmno\n"))
	if err != nil {
		return
	}

	// Output:
	// abc
	// adefghi
	// ajkl
	// amno
}
