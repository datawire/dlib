package dutil

import (
	"context"
	"net"
	"net/http"

	"github.com/datawire/dlib/dcontext"
)

func httpWithContext(ctx context.Context, server *http.Server, fn func() error) error {
	server.BaseContext = func(_ net.Listener) context.Context { return dcontext.HardContext(ctx) }
	serverCh := make(chan error)
	go func() {
		serverCh <- fn()
	}()
	select {
	case err := <-serverCh:
		return err
	case <-ctx.Done():
		return server.Shutdown(dcontext.HardContext(ctx))
	}
}

// ListenAndServeHTTPWithContext runs server.ListenAndServe() on an http.Server, but properly calls
// server.Shutdown when the Context is canceled.
//
// It obeys hard/soft cancellation as implemented by dcontext.WithSoftness; it calls
// server.Shutdown() when the soft Context is canceled, and the hard Context being canceled causes
// the .Shutdown() to hurry along and kill any live requests and return, instead of waiting for them
// to be completed gracefully.
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
func ServeHTTPWithContext(ctx context.Context, server *http.Server, listener net.Listener) error {
	return httpWithContext(ctx, server,
		func() error { return server.Serve(listener) })
}

// ServeHTTPSWithContext runs server.ServeTLS() on an http.Server, but properly calls
// server.Shutdown when the Context is canceled.
//
// It obeys hard/soft cancellation as implemented by dcontext.WithSoftness; it calls
// server.Shutdown() when the soft Context is canceled, and the hard Context being canceled causes
// the .Shutdown() to hurry along and kill any live requests and return, instead of waiting for them
// to be completed gracefully.
func ServeHTTPSWithContext(ctx context.Context, server *http.Server, ln net.Listener, certFile, keyFile string) error {
	return httpWithContext(ctx, server,
		func() error { return server.ServeTLS(ln, certFile, keyFile) })
}
