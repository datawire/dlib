package dutil

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
)

type connContextFn func(ctx context.Context, c net.Conn) context.Context

func concatConnContext(fns ...connContextFn) connContextFn {
	return func(ctx context.Context, c net.Conn) context.Context {
		for _, fn := range fns {
			if fn != nil {
				ctx = fn(ctx, c)
				if ctx == nil {
					// This is the same check that http.Server.Serve does.
					panic("ConnContext returned nil")
				}
			}
		}
		return ctx
	}
}

// If you find it nescessary to edit this function, then you should probably also edit the example
// in `dcontext/hardsoft_example_test.go`.
func httpWithContext(ctx context.Context, server *http.Server, fn func() error) error {
	if server.BaseContext != nil {
		pc, _, _, _ := runtime.Caller(1)
		qname := runtime.FuncForPC(pc).Name()
		dot := strings.LastIndex(qname, ".")
		name := qname[dot+1:]
		return errors.Errorf("it is invalid to call %s with the Server.BaseContext set", name)
	}

	// Set up a cancel to ensure that we don't leak a live Context to stray goroutines.
	hardCtx, hardCancel := context.WithCancel(dcontext.HardContext(ctx))
	defer hardCancel()

	// Regardless of if you use dcontext, if you're using Contexts at all, then you should
	// always set `.BaseContext` on your `http.Server`s so that your HTTP Handler receives a
	// request object that has `Request.Context()` set correctly.
	server.BaseContext = func(_ net.Listener) context.Context {
		// We use the hard Context here instead of the soft Context so
		// that in-progress requests don't get interrupted when we enter
		// the shutdown grace period.
		return hardCtx
	}
	server.ConnContext = concatConnContext(
		func(ctx context.Context, c net.Conn) context.Context {
			return dgroup.WithGoroutineName(ctx, "/"+c.LocalAddr().String())
		},
		server.ConnContext,
	)
	if server.ErrorLog == nil {
		server.ErrorLog = dlog.StdLogger(ctx, dlog.LogLevelError)
	}

	serverCh := make(chan error)
	go func() {
		serverCh <- fn()
		close(serverCh)
	}()

	var err error
	select {
	case err = <-serverCh:
		// The server quit on its own.
	case <-ctx.Done():
		// A soft shutdown has been initiated; call server.Shutdown().
		err = server.Shutdown(hardCtx)

		// If the hardCtx becomes Done before server shuts down, then server.Shutdown()
		// simply returns early, without doing any more-aggressive shutdown logic.  So in
		// that case, we'll need to call server.Close() ourselves to propagate the hard
		// shutdown.
		_ = server.Close()
		<-serverCh // Don't leak the channel
	}

	return err
}

// ListenAndServeHTTPWithContext runs server.ListenAndServe() on an http.Server, but properly calls
// server.Shutdown when the Context is canceled.
//
// It obeys hard/soft cancellation as implemented by dcontext.WithSoftness; it calls
// server.Shutdown() when the soft Context is canceled, and the hard Context being canceled causes
// the .Shutdown() to hurry along and kill any live requests and return, instead of waiting for them
// to be completed gracefully.
//
// It is invalid to call ListenAndServeHTTPWithContext with server.BaseContext set; the passed-in
// Context is the base Context.
func ListenAndServeHTTPWithContext(ctx context.Context, server *http.Server) error {
	return httpWithContext(ctx, server,
		server.ListenAndServe)
}

// ListenAndServeHTTPSWithContext runs server.ListenAndServeTLS() on an http.Server, but properly
// calls server.Shutdown when the Context is canceled.
//
// It obeys hard/soft cancellation as implemented by dcontext.WithSoftness; it calls
// server.Shutdown() when the soft Context is canceled, and the hard Context being canceled causes
// the .Shutdown() to hurry along and kill any live requests and return, instead of waiting for them
// to be completed gracefully.
//
// It is invalid to call ListenAndServeHTTPSWithContext with server.BaseContext set; the passed-in
// Context is the base Context.
func ListenAndServeHTTPSWithContext(ctx context.Context, server *http.Server, certFile, keyFile string) error {
	return httpWithContext(ctx, server,
		func() error { return server.ListenAndServeTLS(certFile, keyFile) })
}

// ServeHTTPWithContext(ln) runs server.Serve(ln) on an http.Server, but properly calls
// server.Shutdown when the Context is canceled.
//
// It obeys hard/soft cancellation as implemented by dcontext.WithSoftness; it calls
// server.Shutdown() when the soft Context is canceled, and the hard Context being canceled causes
// the .Shutdown() to hurry along and kill any live requests and return, instead of waiting for them
// to be completed gracefully.
//
// It is invalid to call ServeHTTPWithContext with server.BaseContext set; the passed-in Context is
// the base Context.
func ServeHTTPWithContext(ctx context.Context, server *http.Server, ln net.Listener) error {
	return httpWithContext(ctx, server,
		func() error { return server.Serve(ln) })
}

// ServeHTTPSWithContext runs server.ServeTLS() on an http.Server, but properly calls
// server.Shutdown when the Context is canceled.
//
// It obeys hard/soft cancellation as implemented by dcontext.WithSoftness; it calls
// server.Shutdown() when the soft Context is canceled, and the hard Context being canceled causes
// the .Shutdown() to hurry along and kill any live requests and return, instead of waiting for them
// to be completed gracefully.
//
// It is invalid to call ServeHTTPSWithContext with server.BaseContext set; the passed-in Context is
// the base Context.
func ServeHTTPSWithContext(ctx context.Context, server *http.Server, ln net.Listener, certFile, keyFile string) error {
	return httpWithContext(ctx, server,
		func() error { return server.ServeTLS(ln, certFile, keyFile) })
}
