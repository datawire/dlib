package dhttp

import (
	"context"
	"net"
	"net/http"
	"sync"
)

type connContextKey struct{}

// configureHijackTracking configures (mutates) an *http.Server to provide slightly better tracking
// of Hijack()ed connections.  It returns a 'close' function that closes all active hijacked
// connections (you should call this when you call server.Close), and a 'wait' function that blocks
// until all of the workers have quit (you should call this when you call server.Shutdown).
//
// This wraps the server.Handler, so it should be called *after* setting up any Handler that might
// hijack connections.
func configureHijackTracking(server *http.Server) (close func(), wait func()) {
	var wg sync.WaitGroup

	var mu sync.Mutex                            // protects 'hijackedConns'
	hijackedConns := make(map[net.Conn]struct{}) // protected by 'mu'

	// Make a note of it whenever a connection gets hijacked.
	origConnState := server.ConnState
	server.ConnState = func(conn net.Conn, state http.ConnState) {
		if origConnState != nil {
			origConnState(conn, state)
		}
		if state == http.StateHijacked {
			mu.Lock()
			hijackedConns[conn] = struct{}{}
			mu.Unlock()
		}
	}

	// Pack the net.Conn in to the context, so that we can access it below.
	server.ConnContext = concatConnContext(
		func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, connContextKey{}, c)
		},
		server.ConnContext,
	)

	// (1) Make a note of it whenever a hijacked connection's worker returns, so that we don't
	// need to keep track of that connection forever. (2) Keep track of whether there are still
	// outstanding workers.
	origHandler := server.Handler
	if origHandler == nil {
		origHandler = http.DefaultServeMux
	}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wg.Add(1)
		defer wg.Done()
		defer func() {
			mu.Lock()
			defer mu.Unlock()
			conn := r.Context().Value(connContextKey{}).(net.Conn)
			delete(hijackedConns, conn)
		}()
		origHandler.ServeHTTP(w, r)
	})

	closeHijacked := func() {
		mu.Lock()
		defer mu.Unlock()
		for conn := range hijackedConns {
			conn.Close()
			delete(hijackedConns, conn)
		}
	}

	return closeHijacked, wg.Wait
}
