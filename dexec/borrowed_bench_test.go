// MODIFIED: META: This file is copied verbatim from Go 1.15.14 os/exec/bench_test.go,
// MODIFIED: META: except for lines marked "MODIFIED".

// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dexec // MODIFIED: FROM: package exec

import (
	"testing"

	"github.com/datawire/dlib/dlog" // MODIFIED: ADDED
)

func BenchmarkExecHostname(b *testing.B) {
	ctx := dlog.NewTestContext(b, false) // MODIFIED: ADDED
	b.ReportAllocs()
	path, err := LookPath("hostname")
	if err != nil {
		b.Fatalf("could not find hostname: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := CommandContext(ctx, path).Run(); err != nil { // MODIFIED: FROM: if err := Command(path).Run(); err != nil {
			b.Fatalf("hostname: %v", err)
		}
	}
}
