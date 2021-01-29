package dutil

import (
	"github.com/datawire/dlib/derror"
)

// PanicToError is a legacy alias for derror.PanicToError.
//
// Note: We use a variable here (instead of wrapping the function) in order to avoid adding an extra
// entry to the stacktrace.
var PanicToError = derror.PanicToError
