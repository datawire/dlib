package dutil_test

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dlog"
	"github.com/datawire/dlib/dutil"
)

// TestHTTPHardShutdown checks to make sure that the TCP connection gets forcefully closed when the
// server gets a hard-shutdown.
func TestHTTPHardShutdown(t *testing.T) {
	ctx, hardCancel := context.WithCancel(dlog.NewTestContext(t, false))
	defer hardCancel()
	ctx = dcontext.WithSoftness(ctx)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if listener, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			t.Fatalf("httptest: failed to listen on a port: %v", err)
		}
	}

	url := "http://" + listener.Addr().String()
	sRequestReceived := make(chan struct{})
	sRequestCanceled := make(chan struct{})
	sRequestFinished := make(chan struct{})
	cRequestFinished := make(chan struct{})
	sExited := make(chan struct{})

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(sRequestReceived)

			<-r.Context().Done()
			close(sRequestCanceled)

			// Waiting for <-cRequestFinished is important; we need to make sure that a
			// hard shutdown is triggerd when handlers hang.
			<-cRequestFinished
			close(sRequestFinished)
		}),
	}

	go func() {
		assert.Error(t, dutil.ServeHTTPWithContext(ctx, srv, listener))
		close(sExited)
	}()
	go func() {
		resp, err := http.Get(url)
		// `err != nil` is important; if the request got interrupted then it's important
		// that it isn't just interpretted as a truncated response, and is actually an
		// error.
		assert.Error(t, err)
		assert.Nil(t, resp)
		close(cRequestFinished)
	}()

	<-sRequestReceived
	hardCancel()
	<-sRequestCanceled
	<-cRequestFinished
	<-sExited
	<-sRequestFinished
}
