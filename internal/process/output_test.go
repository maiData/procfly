package process_test

import (
	"fmt"
	"os"

	"github.com/emm035/procfly/internal/process"
)

func ExampleMuxWriter() {
	pwf := process.NewMuxWriter(os.Stdout)

	prw1 := pwf.Writer("a")
	fmt.Fprint(prw1, "bc\ndef")
	fmt.Fprintln(prw1, "ghi")

	prw2 := pwf.Writer("ab")
	fmt.Fprintln(prw2, "jkl\nmno")

	fmt.Fprintln(prw1, "pqr")

	// Output:
	// a | bc
	// a | def
	// a | ghi
	// ab | jkl
	// ab | mno
	// a  | pqr
}
