package derror_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/derror"
)

func TestMultiError(t *testing.T) {
	type testcase struct {
		Input          derror.MultiError
		ExpectedString string
		ExpectedIs     []error
		ExpectedIsNot  []error
	}
	testcases := map[string]testcase{
		"nil": {
			Input:          nil,
			ExpectedString: "(0 errors; BUG: this should not be reported as an error)",
		},
		"zero": {
			Input:          derror.MultiError{},
			ExpectedString: "(0 errors; BUG: this should not be reported as an error)",
		},
		"one": {
			Input: derror.MultiError{
				fmt.Errorf("just my luck: %w", io.EOF),
			},
			ExpectedString: "just my luck: EOF",
			ExpectedIs:     []error{io.EOF},
			ExpectedIsNot:  []error{errors.New("EOD")},
		},
		"newlines": {
			Input: derror.MultiError{
				errors.New("first error"),
				errors.New("second\nerror"),
				errors.New("third error"),
				errors.New("fourth error"),
				errors.New("fifth error"),
				errors.New("sixth error"),
				errors.New("7th error"),
				errors.New("8th error"),
				errors.New("9th error"),
				errors.New("tenth\nerror"),
			},
			ExpectedString: `10 errors:
 1. first error
 2. second
    error
 3. third error
 4. fourth error
 5. fifth error
 6. sixth error
 7. 7th error
 8. 8th error
 9. 9th error
 10. tenth
     error`,
		},
	}
	for tcName, tcData := range testcases {
		tcData := tcData
		t.Run(tcName, func(t *testing.T) {
			assert.Equal(t, tcData.ExpectedString, tcData.Input.Error())
			for i, err := range tcData.ExpectedIs {
				assert.Truef(t, errors.Is(tcData.Input, err), ".ExpectedIs[i]", i)
			}
			for i, err := range tcData.ExpectedIsNot {
				assert.Falsef(t, errors.Is(tcData.Input, err), ".ExpectedIsNot[i]", i)
			}
		})
	}
}
