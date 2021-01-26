package dutil

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/dlib/dhttp"
)

func serverConfig(server *http.Server) (*dhttp.ServerConfig, error) {
	if server.BaseContext != nil {
		pc, _, _, _ := runtime.Caller(1)
		qname := runtime.FuncForPC(pc).Name()
		dot := strings.LastIndex(qname, ".")
		name := qname[dot+1:]
		return nil, errors.Errorf("it is invalid to call %s with the Server.BaseContext set", name)
	}

	config := &dhttp.ServerConfig{
		Handler:           server.Handler,
		TLSConfig:         server.TLSConfig,
		ReadTimeout:       server.ReadTimeout,
		ReadHeaderTimeout: server.ReadHeaderTimeout,
		IdleTimeout:       server.IdleTimeout,
		MaxHeaderBytes:    server.MaxHeaderBytes,
		ConnState:         server.ConnState,
		ConnContext:       server.ConnContext,
		TLSNextProto:      server.TLSNextProto,
		ErrorLog:          server.ErrorLog,
	}

	return config, nil
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
	sc, err := serverConfig(server)
	if err != nil {
		return err
	}
	return sc.ListenAndServe(ctx, server.Addr)
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
	sc, err := serverConfig(server)
	if err != nil {
		return err
	}
	return sc.ListenAndServeTLS(ctx, server.Addr, certFile, keyFile)
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
	sc, err := serverConfig(server)
	if err != nil {
		return err
	}
	return sc.Serve(ctx, ln)
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
//
// ServeHTTPSWithContext always closes the Listener before returning (this is slightly different
// than *http.Server.ServeTLS, which does not close the Listener if returning early during setup due
// to being passed invalid cert or key files).
func ServeHTTPSWithContext(ctx context.Context, server *http.Server, ln net.Listener, certFile, keyFile string) error {
	sc, err := serverConfig(server)
	if err != nil {
		return err
	}
	return sc.ServeTLS(ctx, ln, certFile, keyFile)
}
