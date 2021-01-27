package dhttp

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

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

// ServerConfig is a mostly-drop-in replacement for net/http.Server.
type ServerConfig struct {
	// These fields mimic exactly mimic http.Server; see the documentation there.
	Handler           http.Handler
	TLSConfig         *tls.Config
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ConnState         func(net.Conn, http.ConnState)
	ConnContext       func(ctx context.Context, c net.Conn) context.Context
	TLSNextProto      map[string]func(*http.Server, *tls.Conn, http.Handler)

	// ErrorLog (mostly mimicking http.Server.ErrorLog) specifies an optional logger for errors
	// accepting connections, unexpected behavior from handlers, and underlying file-system
	// errors.
	//
	// If nil, logging is done via the dlog with LogLevelError with the Context passed to the
	// Serve function (this is different than http.Server.ErrorLog, which would use the log
	// package's standard logger).
	ErrorLog *log.Logger
}

// If you find it nescessary to edit this function, then you should probably also edit the example
// in `dcontext/hardsoft_example_test.go`.
func (sc *ServerConfig) serve(ctx context.Context, serveFn func(*http.Server) error) error {
	// Set up a cancel to ensure that we don't leak a live Context to stray goroutines.
	hardCtx, hardCancel := context.WithCancel(dcontext.HardContext(ctx))
	defer hardCancel()

	server := &http.Server{
		// Pass along the verbatim fields
		Handler:           sc.Handler,
		TLSConfig:         sc.TLSConfig, // don't worry about deep-copying the TLS config, net/http will do it
		ReadTimeout:       sc.ReadTimeout,
		ReadHeaderTimeout: sc.ReadHeaderTimeout,
		IdleTimeout:       sc.IdleTimeout,
		MaxHeaderBytes:    sc.MaxHeaderBytes,
		ConnState:         sc.ConnState,
		ConnContext: concatConnContext(
			func(ctx context.Context, c net.Conn) context.Context {
				return dgroup.WithGoroutineName(ctx, "/"+c.LocalAddr().String())
			},
			sc.ConnContext,
		),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), len(sc.TLSNextProto)), // deep-copy below
		ErrorLog:     sc.ErrorLog,

		// Regardless of if you use dcontext, if you're using Contexts at all, then you should
		// always set `.BaseContext` on your `http.Server`s so that your HTTP Handler receives a
		// request object that has `Request.Context()` set correctly.
		BaseContext: func(_ net.Listener) context.Context {
			// We use the hard Context here instead of the soft Context so
			// that in-progress requests don't get interrupted when we enter
			// the shutdown grace period.
			return hardCtx
		},
	}
	for k, v := range sc.TLSNextProto {
		server.TLSNextProto[k] = v
	}
	if server.ErrorLog == nil {
		server.ErrorLog = dlog.StdLogger(ctx, dlog.LogLevelError)
	}

	serverCh := make(chan error)
	go func() {
		serverCh <- serveFn(server)
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

// Serve accepts incoming connection on the Listener ln, creating a new service goroutine for each.
// The service goroutines read requests and call sc.Handler to reply to them.
//
// Serve always closes the Listener before returning.
func (sc *ServerConfig) Serve(ctx context.Context, ln net.Listener) error {
	return sc.serve(ctx, func(srv *http.Server) error { return srv.Serve(ln) })
}

// Serve accepts incoming connection on the Listener ln, creating a new service goroutine for each.
// The service goroutines perform TLS setup, and then read requests and call sc.Handler to reply to
// them.
//
// Files containing a certificate and matching private key for the server must be provided if
// neither the Server's TLSConfig.Certificates nor TLSConfig.GetCertificate are populated.  If the
// certificate is signed by a certificate authority, the certFile should be the concatenation of the
// server's certificate, any intermediates, and the CA's certificate.
func (sc *ServerConfig) ServeTLS(ctx context.Context, ln net.Listener, certFile, keyFile string) error {
	return sc.serve(ctx, func(srv *http.Server) error { return srv.ServeTLS(ln, certFile, keyFile) })
}

// ListenAndServeTLS is like Serve, but rather than taking an existing Listener object, it takes a
// TCP address to listen on.  If an empty address is given, then ":http" is used.
func (sc *ServerConfig) ListenAndServe(ctx context.Context, addr string) error {
	if addr == "" {
		addr = ":http"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return sc.Serve(ctx, ln)
}

// ListenAndServeTLS is like ServeTLS, but rather than taking an existing cleartext Listener object,
// it takes a TCP address to listen on.  If an empty address is given, then ":https" is used.
func (sc *ServerConfig) ListenAndServeTLS(ctx context.Context, addr, certFile, keyFile string) error {
	if addr == "" {
		addr = ":https"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// Make sure we close the Listener before we return; the underlying srv.ServeTLS won't close
	// it if it returns early during setup due to being passed invalid cert or key files.
	defer ln.Close()

	return sc.ServeTLS(ctx, ln, certFile, keyFile)
}
