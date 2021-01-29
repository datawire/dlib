// Package dhttp is a simple production-ready HTTP server library for the 2020s.
//
// In his famous talk "Simplicity is Complicated"[1], Rob Pike talks about how even the most simple
// server using net/http.Server is already production-ready (and about how complicated it was for
// the Go designers to achieve that).  That was true in November 2015, when the talk was given.
// However, what "production-ready" means and what "simple" means have both changed since then.
//
// "Production-ready" has changed; it now includes HTTP/2 support; HTTP/2 had just been finalized a
// few months prior to the talk, and so wasn't something expected for production yet.  Some
// quite-good work has since gone in to net/http.Server to support HTTP/2 over TLS, but HTTP/2 over
// cleartext is forced to be left out because of backward-compatibility concerns; leaving the user
// to have to bolt it on themselves using golang.org/x/net/http2/h2c.  Cleartext users of
// net/http.Server must now choose between "production-ready" and "simplicity".  And even if the
// user does decide to sacrifice that simplicity, the solution cannot be called really be
// production-ready, since with x/net/http2/h2c alone cleartext HTTP/2 connections are not properly
// shut down when shutting down the server.
//
// "Simple" has changed; in August of the next year, the Go standard library gained "context", a
// unified mechanism for several things, most notably here for managing the lifecycle of processes.
// Simple now means using Contexts for the lifecycle.  In order to tack on Context support to
// net/http.Server, it gained a complex and confusing relationship between "Shutdown", "Close", and
// "BaseContext".  While net/http.Server has gained Context support, because of
// backward-compatibility concerns, it could not do it in a way that achieved simplicity.
//
// This package provides a simple production-ready HTTP server library, for the meaning that those
// words have going in to the 2020s.  To accomplish this, it makes breaking changes from
// net/http.Server, but keeps them to a minimum; it should still be familiar and comfortable to
// those who are already used to net/http.Server.  Fret not about throwing away all of the
// engineering that went in to net/http.Server; this package still uses net/http.Server internally.
//
// [1]: https://www.youtube.com/watch?v=rFejpH_tAHM and
// https://talks.golang.org/2015/simplicity-is-complicated.slide
package dhttp

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"

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

type testHookContextKey struct{}

// ServerConfig is a mostly-drop-in replacement for net/http.Server.
//
// This is better than http.Server because:
//
//  - It natively supports "h2c" (HTTP/2 over cleartext)
//  - Its "h2c" support is better than http.Server with golang.org/x/net/http2/h2c.NewHandler,
//    because it properly shuts down idle h2c connections when server.Shutdown is called, rather
//    than allowing h2c connections to sit around forever.
//  - It uses Context cancellation as a simple and composable shutdown mechanism; supporting
//    dcontext hard/soft contexts, rather than awkward relationship between the Shutdown and Close
//    methods.
//  - When shutting down, it properly blocks when waiting for the workers of hijacked connections.
//  - If you use dlog, you don't have to manually configure the logging for the server to do the
//    right thing.
//
// Breaking changes from http.Server to ServerConfig that will stop your old code from compiling:
//
//  - Obviously, the name is different.
//  - The "Addr" member field is removed; it is replaced by an "addr" argument to the
//    "ListenAndServe(TLS)?" methods.
//  - The "BaseContext" member field is removed; it is replaced by a "ctx" argument to the
//    "(ListenAnd)?Serve(TLS?)" methods.
//  - The "RegisterOnShutdown" is removed; it is replaced by an "OnShutdown" member field.
//  - The "SetKeepAlivesEnabled", "Shutdown", and "Close" methods are removed; they are conceptually
//    replaced by using Context cancellation for lifecycle management.  Use dcontext soft
//    cancellation for the graceful shutdown that "Shutdown" allowed.
//
// Breaking changes from http.Server to ServerConfig that will maybe make your old code incorrect:
//
//  - The semantics of the "TLSNextProto" member field are slightly different.
//  - The semantics of the "Error" member field are slightly different.
//  - The structure is deep-copied by each of the "(ListenAnd)?Serve(TLS?)" methods; mutating the
//    config structure while a server is running will not affect the running server.
//  - HTTP/2 support (both "h2" and "h2c") is built-in, so if your code configures HTTP/2 manually,
//    you're going to need to set "DisableHTTP2: true" to stop ServerConfig from stomping over your
//    code's work.
//
// Arguably-breaking changes from http.Server that to ServerConfig that I'd say are bugfixes, but
// could conceivably[1] make someone's old code incorrect:
//
//  - *http.Server.ServeTLS won't close the Listener if .ServeTLS returns early during setup due to
//    having been passed invalid cert or key files; ServerConfig.ServeTLS will always close the
//    Listener before returning; matching the "Serve" method.
//
// The reason for creating a new type and having breaking changes (rather than writing a few utility
// functions that take an *http.Server as an argument) is that it became increasingly clear that the
// lifecycle of a running server is tied to the lifecycle of the *http.Server object, while we've
// grown a standard "Context" lifecycle system that wants to tie the lifecycle to the function call,
// Rather than to the object.  This divorce between the running-server lifecycle and the object is
// embodied in the name of the type; when the lifecycle of the object was the lifecycle of the
// server, the object *was* the server, so it was named *http.Server; now the lifecycle is divorced
// from that, and the object is just configuration for the server, so it is named *ServerConfig.
//
// [1]: https://xkcd.com/1172/
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

	// TLSNextProto (mostly mimicking http.Server.TLSNextProto) optionally specifies a function
	// to take over ownership of the provided TLS connection when an ALPN protocol upgrade has
	// occurred.  The map key is the protocol name negotiated.  The Handler argument should be
	// used to handle HTTP requests and will initialize the Request's TLS and RemoteAddr if not
	// already set.  The connection is automatically closed when the function returns.
	//
	// If you provide an "h2" entry, it will be forcefully overwritten unless DisableHTTP2 is
	// true (this is different than http.Server.TLSNextProto, which only enables HTTP/2 if
	// TLSNextProto is nil).
	TLSNextProto map[string]func(*http.Server, *tls.Conn, http.Handler)

	// ErrorLog (mostly mimicking http.Server.ErrorLog) specifies an optional logger for errors
	// accepting connections, unexpected behavior from handlers, and underlying file-system
	// errors.
	//
	// If nil, logging is done via the dlog with LogLevelError with the Context passed to the
	// Serve function (this is different than http.Server.ErrorLog, which would use the log
	// package's standard logger).
	ErrorLog *log.Logger

	// DisableHTTP2 controls whether both "h2" (HTTP/2 over TLS) and "h2c" (HTTP/2 over
	// cleartext) are enabled or disabled.
	//
	// (This is not in http.Server at all.)
	DisableHTTP2 bool

	// HTTP2Config contains the HTTP/2-specific configuration (except for whether HTTP/2 is
	// enabled at all; use DisableHTTP2 for that).  HTTP2Config may be nil, and HTTP/2 will
	// still be enabled.
	//
	// (This is not in http.Server at all.)
	HTTP2Config *http2.Server

	// OnShutdown is an array of functions that are each called once when shutdown is initiated.
	// Use this when hijacking connections; your OnShutdown should notify your hijacking Handler
	// to that a graceful shutdown has been initiated, and your Handler should respond by
	// closing any idle connections.  This is used instead of dcontext soft Context cancellation
	// because the Context should very much still be fully alive for any in-progress requests on
	// that connection, and not soft-canceled; this is even softer than a dcontext soft cancel.
	//
	// (This replaces the RegisterOnShutdown method of *http.Server.)
	OnShutdown []func()
}

func (sc *ServerConfig) serve(ctx context.Context, serveFn func(*http.Server) error) error {
	// Part 1: Set up a cancel to ensure that we don't leak a live Context to stray goroutines.
	hardCtx, hardCancel := context.WithCancel(dcontext.HardContext(ctx))
	defer hardCancel()

	// Part 2: Instantiate the basic *http.Server.
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
	for _, onShutdown := range sc.OnShutdown {
		server.RegisterOnShutdown(onShutdown)
	}

	// Part 3: Configure HTTP/2.
	//
	// Note that this still has a "gotcha" with h2c connections not being properly tracked
	// because they show as hijacked (see the doc comment on configureHTTP2).  We'll address
	// that below with configureHijackTracking.
	if !sc.DisableHTTP2 {
		cfg := sc.HTTP2Config
		if cfg != nil {
			// shallow copy (there's nothing deep inside of it)
			_cfg := *cfg
			cfg = &_cfg
		}
		if err := configureHTTP2(server, cfg); err != nil {
			return err
		}
	}

	// Part 4: Configure tracking of hijacked connections.
	//
	// This is good in general, but really the motivating reason for it is for h2c connections
	// (see above).  This must be called *after* configureHTTP2.
	closeHijacked, waitHijacked := configureHijackTracking(server)

	// Part n: Testing
	if untyped := ctx.Value(testHookContextKey{}); untyped != nil {
		testHook := untyped.(func(http.Handler) http.Handler)
		server.Handler = testHook(server.Handler)
	}

	// Part 5: Actually run the thing.

	serverCh := make(chan error)
	go func() {
		serverCh <- serveFn(server)
		close(serverCh)
	}()

	var err error
	select {
	case err = <-serverCh:
		// The server encountered an error and bailed on its own.  Tell any hijacked
		// connections to also bail.
		hardCancel()
		_ = server.Shutdown(hardCtx)
	case <-ctx.Done():
		// A soft shutdown has been initiated; call server.Shutdown().
		err = server.Shutdown(hardCtx)
		<-serverCh // server returns immediately upon calling .Shutdown; don't leak the channel
	}

	// At this point, everything managed by the http.Server has finished, but hijacked
	// connections may still be going.  We don't want to forcfully kill them if we haven't
	// actually had a hard shutdown triggered yet.  So wait for that to happen.
	workersDoneCh := make(chan struct{})
	go func() {
		waitHijacked()
		close(workersDoneCh)
	}()
	select {
	case <-hardCtx.Done():
		if err == nil {
			err = hardCtx.Err()
		}
	case <-workersDoneCh:
	}

	// Trigger the hard shutdown.  This is probably not nescessary in the <-workersDoneCh case,
	// but let's do it in both cases, just to be safe.
	//
	// If the hardCtx becomes Done before server shuts down, then server.Shutdown() simply
	// returns early, without doing any more-aggressive shutdown logic.  So we really do need to
	// trigger the hard shutdown ourselves.
	//
	// Do the hardCancel *after* the "close" calls so that any truncated responses aren't
	// mistakenly treated as complete.
	_ = server.Close()
	closeHijacked()
	hardCancel()

	// Wait for the workers to shut down.  This is normally done by server.Shutdown, but (1)
	// server.Shutdown might have bailed early, and (2) server.Shutdown ignores hijacked
	// connections.
	<-workersDoneCh

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
//
// ServeTLS always closes the Listener before returning (this is slightly different than
// *http.Server.ServeTLS, which does not close the Listener if returning early during setup due to
// being passed invalid cert or key files).
func (sc *ServerConfig) ServeTLS(ctx context.Context, ln net.Listener, certFile, keyFile string) error {
	// Make sure we close the Listener before we return; the underlying srv.ServeTLS won't close
	// it if it returns early during setup due to being passed invalid cert or key files.
	defer ln.Close()

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

	// If you're comparing this method to *http.Server.ListenAndServeTLS, then you're probably
	// thinking "Don't we need a `defer ln.Close()` here!?" (and also probably wondering why
	// *http.Server needs that statement).  The answer is "no, we don't need it", because we
	// handle that in ServeTLS instead (and see the comments there about why it's nescessary).

	return sc.ServeTLS(ctx, ln, certFile, keyFile)
}
