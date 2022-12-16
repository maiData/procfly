package process_test

import (
	"os"

	"github.com/emm035/procfly/internal/process"
)

func ExampleMuxWriterFactory() {
	pwf := process.NewMuxWriter(os.Stdout, 1)
	prw := pwf.Writer("a")

	_, err := prw.Write([]byte("bc\ndef"))
	if err != nil {
		return
	}

	_, err = prw.Write([]byte("ghi\n"))
	if err != nil {
		return
	}

	prw = pwf.Writer("b")

	_, err = prw.Write([]byte("jkl\nmno\n"))
	if err != nil {
		return
	}

	// Output:
	// a | bc
	// a | defghi
	// b | jkl
	// b | mno
}
