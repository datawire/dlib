// Copyright 2021 Datawire. All rights reserved.
//
// This file is based on Go 1.17.1 sync/cond.go.
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE. file.

package dsync

import (
	"sync/atomic"
	"unsafe"
)

// noCopyRuntime may be embedded into structs which must not be copied after the first use, and then
// .check() called to detect copies at runtime.
type noCopyRuntime uintptr

// return whether the check is OK
func (c *noCopyRuntime) check() bool {
	if atomic.LoadUintptr((*uintptr)(c)) != uintptr(unsafe.Pointer(c)) &&
		!atomic.CompareAndSwapUintptr((*uintptr)(c), 0, uintptr(unsafe.Pointer(c))) &&
		atomic.LoadUintptr((*uintptr)(c)) != uintptr(unsafe.Pointer(c)) {
		// it was copied!
		return false
	}
	// it was not copied
	return true
}

// noCopyVet may be embedded into structs which must not be copied after the first use, in order for
// `go vet -copylocks` to detect the copy.
//
// See https://golang.org/issues/8005#issuecomment-190753527 for details.
type noCopyVet struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopyVet) Lock()   {}
func (*noCopyVet) Unlock() {}
