package dlib_test

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/derror"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
	"github.com/datawire/dlib/dtime"
)

// This is an example main() program entry-point that shows how all the pieces of dlib can fit
// together and complement each other.
func main() {
	// Start with the background Context as the root Context.
	ctx := context.Background()

	// The default backend for dlog is pretty good, but for the sake of example, let's customize
	// it a bit.
	ctx = dlog.WithLogger(ctx, func() dlog.Logger {
		// Let's have the backend be logrus.  The default backend is already logrus, but
		// ours will be customized.
		logrusLogger := logrus.New()
		// The dlog default is InfoLevel; let's crank it up to DebugLevel.
		logrusLogger.Level = logrus.DebugLevel
		// Now turn that in to a dlog.Logger backend, so we can pass it to dlog.WithLogger.
		return dlog.WrapLogrus(logrusLogger)
	}())

	// We're going to be doing several tasks in parallel, so we'll use "dgroup" to manage our
	// group of goroutines.
	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		// Enable signal handling for graceful shutdown.  The user can stop the program by
		// sending it SIGINT with Ctrl-C, and that will start a graceful shutdown.  If that
		// graceful shutdown takes too long, and the user hits Ctrl-C again, then it will
		// start a not-so-graceful shutdown.
		//
		// This shutdown will be signaled to the worker goroutines through the Context that
		// gets passed to them.  The mechanism by which the Context signals both graceful
		// and not-so-graceful shutdown is what "dcontext" is for.
		EnableSignalHandling: true,
	})

	// One of those tasks will be running an HTTP server.
	grp.Go("http", func(ctx context.Context) error {
		// We'll be using a *dhttp.ServerConfig instead of an *http.Server, but it works
		// very similarly to *http.Server, everything else in the stdlib net/http package is
		// still valid; we'll still be using plain-old http.ResponseWriter and *http.Request
		// and http.HandlerFunc.
		cfg := &dhttp.ServerConfig{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				dlog.Debugln(r.Context(), "handling HTTP request")
				_, _ = w.Write([]byte("Hello, world!\n"))
			}),
		}
		// ListenAndServe will gracefully shut down according to ctx; we don't need to worry
		// about separately calling .Shutdown() or .Close() like we would for *http.Server
		// (those methods don't even exist on dhttp.ServerConfig).  During a graceful
		// shutdown, it will stop listening and close idle connections, but will wait on any
		// active connections; during a not-so-graceful shutdown it will forcefully close
		// any active connections.
		//
		// If the server itself needs to log anything, it will use dlog according to ctx.
		// The Request.Context() passed to the Handler function will inherit from ctx, and
		// so the Handler will also log according to ctx.
		//
		// And, on the end-user-facing side of things, this supports HTTP/2, where
		// *http.Server.ListenAndServe wouldn't.
		return cfg.ListenAndServe(ctx, ":8080")
	})

	// Another task will be running external host operating system commands.
	grp.Go("exec", func(ctx context.Context) error {
		// dexec is *almost* a drop-in replacement for os/exec; the only breaking change is
		// that .Command() doesn't exist anymore, you *must* use the .CommandContext()
		// variant.
		//
		// There are two nice things using dexec instead of os/exec this gets us.
		//
		// Firstly: dexec logs everything read from or written to the command's stdin,
		// stdout, and stderr; logging with dlog according to ctx.  Logging of one of any of
		// these can be opted-out of; see the dexec documentation.
		//
		// Secondly: When the Context signals a shutdown, os/exec responds by stopping the
		// command by sending it the SIGKILL signal.  Now, "sending the SIGKILL signal" is a
		// bit of a misleading phrase, because SIGKILL is never actually sent to the
		// process, it instead tells the operating system to just stop giving the process
		// any CPU time; the process is just abruptly dead.  dexec gives the process a
		// chance to gracefully shutdown by sending it SIGINT for a graceful shutdown, and
		// only sending it SIGKILL for a not-so-graceful shutdown.
		cmd := dexec.CommandContext(ctx, "some", "long-running", "command", "--keep-going")
		return cmd.Run()
	})

	// Another task will be running our own code.
	grp.Go("code", func(ctx context.Context) error {
		dataSourceCh := newDataSource(ctx)
	theloop:
		for {
			select {
			case <-ctx.Done():
				// The channel <-ctx.Done() gets closed when either a graceful or
				// not-so-graceful shutdown is triggerd.
				//
				// So, when the Context signals us to shut down, we'll break out of
				// this `for` loop.
				break theloop
				// ... but until then, read from dataSourceCh, and process the data
				// from it:
			case dat := <-dataSourceCh:
				// The doWorkOnData example function might be a little buggy and
				// might panic().  We don't want that to cause us to stop processing
				// more data early, so we'll use derror to catch any panics and turn
				// them in to useful errors.
				func() {
					defer func() {
						if err := derror.PanicToError(recover()); err != nil {
							// Thanks to PanicToError, err has the panic
							// stacktrace attached to it, which we can
							// show using the "+" modifier to "%v".
							dlog.Errorf(ctx, "doWorkOnData crashed: %+v", err)
						}
					}()
					doWorkOnData(ctx, dat)
				}()
				// We want ensure we wait at least 10 seconds between each time we
				// call DoWorkOnData, but we also don't want to stall a graceful
				// shutdown by 10 seconds, so we'll use dtime.SleepWithContext
				// instead of stdlib time.Sleep.  (We also don't use stdlib
				// time.After, because that would leak channels; we don't want
				// memory leaks!)
				dtime.SleepWithContext(ctx, 10*time.Second)
			}
		}

		// OK, if we're here it's because <-ctx.Done() and we broke out of the `for` loop.
		//
		// We have some cleanup we'd like to do for a graceful shutdown, but we should bail
		// early on that work if a not-so-graceful shutdown is triggered.  If <-ctx.Done()
		// is already closed because of a graceful shutdown, how do we detect when a
		// not-so-graceful shutdown is triggered?
		//
		// Well, ctx is a "soft" Context; signaling a "soft" (graceful) shutdown.  We'll use
		// dcontext.HardContext to get the "hard" Context from it, which signals just a
		// "hard" (not-so-graceful) shutdown.
		ctx = dcontext.HardContext(ctx)
		return gracefulCleanup(ctx)
	})

	if err := grp.Wait(); err != nil {
		dlog.Errorf(ctx, "finished with error: %v", err)
		os.Exit(1)
	}
}

func newDataSource(_ context.Context) <-chan struct{} {
	// Not actually implemented for this example.
	return nil
}

func doWorkOnData(_ context.Context, _ struct{}) {
	// Not actually implemented for this example.
}

func gracefulCleanup(_ context.Context) error {
	// Not actually implemented for this example.
	return nil
}

func Example_main() {
	// An "Example_XXX" function is needed to get this example to show up in the godoc.
	main()
}
