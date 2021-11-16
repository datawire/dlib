package derror_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/derror"
)

func TestMultiError(t *testing.T) {
	err := derror.MultiError{
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
	}
	exp := `10 errors:
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
     error`
	assert.Equal(t, exp, err.Error())
}
