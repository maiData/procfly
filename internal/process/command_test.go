package process_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/maidata/procfly/internal/process"
)

var cases = []struct {
	input    string
	expected process.Command
}{
	{
		input: `command`,
		expected: process.Command{
			Name: "command",
			Args: nil,
		},
	},
	{
		input: `command arg1 arg2 arg3`,
		expected: process.Command{
			Name: "command",
			Args: []string{"arg1", "arg2", "arg3"},
		},
	},
	{
		input: `command "arg1 arg2" arg3`,
		expected: process.Command{
			Name: "command",
			Args: []string{"arg1 arg2", "arg3"},
		},
	},
	{
		input: `command 'arg1 arg2' arg3`,
		expected: process.Command{
			Name: "command",
			Args: []string{"arg1 arg2", "arg3"},
		},
	},
}

func TestCommandUnmarshal(t *testing.T) {
	for i, tc := range cases {
		t.Run("case_"+strconv.Itoa(i), func(t *testing.T) {
			cmd := new(process.Command)
			if err := cmd.UnmarshalText([]byte(tc.input)); err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(*cmd, tc.expected) {
				fmt.Printf("%+v != %+v", *cmd, tc.expected)
				t.Fail()
			}
		})
	}
}
