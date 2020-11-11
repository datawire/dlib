// Package dgroup provides tools for managing groups of goroutines.
//
// The main part of this is Group, but the naming utilities may be
// useful outside of that.
//
// At this point, the limitation of dgroup when compared to supervisor
// is that dgroup does not have a notion of readiness, and does not
// have a notion of dependencies.
package dgroup

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/datawire/ambassador/pkg/dcontext"
	"github.com/datawire/ambassador/pkg/derrgroup"
	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/errutil"
)

// Group is a wrapper around
// github.com/datawire/ambassador/pkg/derrgroup.Group that:
//  - (optionally) handles SIGINT and SIGTERM
//  - (configurable) manages Context for you
//  - (optionally) adds hard/soft cancellation
//  - (optionally) does panic recovery
//  - (optionally) does some minimal logging
//  - (optionally) adds configurable shutdown timeouts
//  - adds a way to call to the parent group
//
// A zero Group is NOT valid; a Group must be created with NewGroup.
type Group struct {
	cfg     GroupConfig
	baseCtx context.Context

	shutdownTimedOut chan struct{}
	waitFinished     chan struct{}
	hardCancel       context.CancelFunc

	workers     *derrgroup.Group
	supervisors sync.WaitGroup
}

func logGoroutineStatuses(
	ctx context.Context,
	heading string,
	printf func(ctx context.Context, format string, args ...interface{}),
	list map[string]derrgroup.GoroutineState,
) {
	printf(ctx, "  %s:", heading)
	names := make([]string, 0, len(list))
	nameWidth := 0
	for name := range list {
		names = append(names, name)
		if len(name) > nameWidth {
			nameWidth = len(name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		printf(ctx, "    %-*s: %s", nameWidth, name, list[name])
	}
}

func logGoroutineTraces(
	ctx context.Context,
	heading string,
	printf func(ctx context.Context, format string, args ...interface{}),
) {
	p := pprof.Lookup("goroutine")
	if p == nil {
		return
	}
	stacktrace := new(strings.Builder)
	if err := p.WriteTo(stacktrace, 2); err != nil {
		return
	}
	printf(ctx, "  %s:", heading)
	for _, line := range strings.Split(strings.TrimSpace(stacktrace.String()), "\n") {
		printf(ctx, "    %s", line)
	}
}

// GroupConfig is a readable way of setting the configuration options
// for NewGroup.
//
// A zero GroupConfig (`dgroup.GroupConfig{}`) should be sane
// defaults.  Because signal handling should only be enabled for the
// outermost group, it is off by default.
//
// TODO(lukeshu): Consider enabling timeouts by default?
type GroupConfig struct {
	// EnableWithSoftness says whether it should call
	// dcontext.WithSoftness() on the Context passed to NewGroup.
	// This should probably NOT be set for a Context that is
	// already soft.  However, this must be set for features that
	// require separate hard/soft cancellation, such as signal
	// handling.  If any of those features are enabled, then it
	// will force EnableWithSoftness to be set.
	EnableWithSoftness   bool
	EnableSignalHandling bool // implies EnableWithSoftness

	// Normally a worker exiting with an error triggers other
	// goroutines to shutdown.  Setting ShutdownOnNonError causes
	// a shutdown to be triggered whenever a goroutine exits, even
	// if it exits without error.
	ShutdownOnNonError bool

	// SoftShutdownTimeout is how long after a soft shutdown is
	// triggered to wait before triggering a hard shutdown.  A
	// zero value means to not trigger a hard shutdown after a
	// soft shutdown.
	//
	// SoftShutdownTimeout implies EnableWithSoftness because
	// otherwise there would be no way of triggering the
	// subsequent hard shutdown.
	SoftShutdownTimeout time.Duration
	// HardShutdownTimeout is how long after a hard shutdown is
	// triggered to wait before forcing Wait() to return early.  A
	// zero value means to not force Wait() to return early.
	HardShutdownTimeout time.Duration

	DisablePanicRecovery bool
	DisableLogging       bool

	WorkerContext func(ctx context.Context, name string) context.Context
}

// NewGroup returns a new Group.
func NewGroup(ctx context.Context, cfg GroupConfig) *Group {
	cfg.EnableWithSoftness = cfg.EnableWithSoftness || cfg.EnableSignalHandling || (cfg.SoftShutdownTimeout > 0)

	ctx, hardCancel := context.WithCancel(ctx)
	var softCancel context.CancelFunc
	if cfg.EnableWithSoftness {
		ctx = dcontext.WithSoftness(ctx)
		ctx, softCancel = context.WithCancel(ctx)
	} else {
		softCancel = hardCancel
	}

	g := &Group{
		cfg: cfg,
		//baseCtx: gets set below,

		shutdownTimedOut: make(chan struct{}),
		waitFinished:     make(chan struct{}),
		hardCancel:       hardCancel,

		workers: derrgroup.NewGroup(softCancel, cfg.ShutdownOnNonError),
		//supervisors: zero value is fine; doesn't need initialize,
	}
	g.baseCtx = context.WithValue(ctx, groupKey{}, g)

	g.launchSupervisors()

	return g
}

// launchSupervisors launches the various "internal" / "supervisor" /
// "helper" goroutines that aren't of concern to the caller of dgroup,
// but are internal to implementing dgroup's various features.
func (g *Group) launchSupervisors() {
	if !g.cfg.DisableLogging {
		g.goSupervisor("shutdown_logger", func(ctx context.Context) error {
			select {
			case <-g.waitFinished:
				// nothing to do
			case <-ctx.Done():
				// log that a shutdown has been triggered
				// be as specific with the logging as possible
				if dcontext.HardContext(ctx) == ctx {
					// no hard/soft distinction
					dlog.Infoln(ctx, "shutting down...")
				} else {
					// there is a hard/soft distinction, check if it's hard or soft
					if dcontext.HardContext(ctx).Err() != nil {
						dlog.Infoln(ctx, "shutting down (not-so-gracefully)...")
					} else {
						dlog.Infoln(ctx, "shutting down (gracefully)...")
						select {
						case <-g.waitFinished:
							// nothing to do
						case <-dcontext.HardContext(ctx).Done():
							dlog.Infoln(ctx, "shutting down (not-so-gracefully)...")
						}
					}
				}
			}
			return nil
		})
	}

	if (g.cfg.SoftShutdownTimeout > 0) || (g.cfg.HardShutdownTimeout > 0) {
		g.goSupervisor("timeout_watchdog", func(ctx context.Context) error {
			if g.cfg.SoftShutdownTimeout > 0 {
				select {
				case <-g.waitFinished:
					// nothing to do
				case <-ctx.Done():
					// soft-shutdown initiated, start the soft-shutdown timeout-clock
					select {
					case <-g.waitFinished:
						// nothing to do, it finished within the timeout
					case <-dcontext.HardContext(ctx).Done():
						// nothing to do, something else went ahead and upgraded
						// this to a hard-shutdown
					case <-time.After(g.cfg.SoftShutdownTimeout):
						// it didn't finish within the timeout,
						// upgrade to a hard-shutdown
						g.hardCancel()
					}
				}
			}
			if g.cfg.HardShutdownTimeout > 0 {
				select {
				case <-g.waitFinished:
					// nothing to do
				case <-dcontext.HardContext(ctx).Done():
					// hard-shutdown initiated, start the hard-shutdown timeout-clock
					select {
					case <-g.waitFinished:
						// nothing to do, it finished within the timeout
					case <-time.After(g.cfg.HardShutdownTimeout):
						close(g.shutdownTimedOut)
					}
				}
			}
			return nil
		})
	}

	if g.cfg.EnableSignalHandling {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		g.goSupervisor("signal_handler", func(ctx context.Context) error {
			<-g.waitFinished
			signal.Stop(sigs)
			close(sigs)
			return nil
		})
		g.goSupervisor("signal_handler", func(ctx context.Context) error {
			i := 0
			for sig := range sigs {
				ctx := WithGoroutineName(ctx, fmt.Sprintf(":%d", i))
				i++

				if ctx.Err() == nil {
					err := errors.Errorf("received signal %v (triggering graceful shutdown)", sig)

					g.goWorkerCtx(ctx, func(_ context.Context) error {
						return err
					})
					<-ctx.Done()

				} else if dcontext.HardContext(ctx).Err() == nil {
					err := errors.Errorf("received signal %v (graceful shutdown already triggered; triggering not-so-graceful shutdown)", sig)

					if !g.cfg.DisableLogging {
						dlog.Errorln(ctx, err)
						logGoroutineStatuses(ctx, "goroutine statuses", dlog.Errorf, g.List())
					}
					g.hardCancel()

				} else {
					err := errors.Errorf("received signal %v (not-so-graceful shutdown already triggered)", sig)

					if !g.cfg.DisableLogging {
						dlog.Errorln(ctx, err)
						logGoroutineStatuses(ctx, "goroutine statuses", dlog.Errorf, g.List())
						logGoroutineTraces(ctx, "goroutine stack traces", dlog.Errorf)
					}
				}
			}
			return nil
		})
	}
}

// Go wraps derrgroup.Group.Go().
//
// Cancellation of the Context should trigger a graceful shutdown.
// Cancellation of the dcontext.HardContext(ctx) of it should trigger
// a not-so-graceful shutdown.
//
// A worker may access its parent group by calling ParentGroup on its
// Context.
func (g *Group) Go(name string, fn func(ctx context.Context) error) {
	g.goWorker(name, fn)
}

// goWorker launches a worker goroutine for the user of dgroup.
func (g *Group) goWorker(name string, fn func(ctx context.Context) error) {
	ctx := WithGoroutineName(g.baseCtx, "/"+name)
	if g.cfg.WorkerContext != nil {
		ctx = g.cfg.WorkerContext(ctx, name)
	}
	g.goWorkerCtx(ctx, fn)
}

// goWorkerCtx() is like goWorker(), except it takes an
// already-created context.
func (g *Group) goWorkerCtx(ctx context.Context, fn func(ctx context.Context) error) {
	g.workers.Go(getGoroutineName(ctx), func() (err error) {
		defer func() {
			if !g.cfg.DisablePanicRecovery {
				if _err := errutil.PanicToError(recover()); _err != nil {
					err = _err
				}
			}
			if !g.cfg.DisableLogging {
				if err == nil {
					dlog.Debugf(ctx, "goroutine %q exited without error", getGoroutineName(ctx))
				} else {
					dlog.Errorf(ctx, "goroutine %q exited with error:", getGoroutineName(ctx), err)
				}
			}
		}()

		return fn(ctx)
	})
}

// goSupervisor launches an "internal" / "supervisor" / "helper"
// goroutine that isn't of concern to the caller of dgroup, but is
// internal to implementing one of dgroup's features.  Put another
// way: they are "systems-logic" goroutines, not "business-logic"
// goroutines.
//
// Compared to normal user-provided "worker" goroutines, these
// "supervisor" goroutines have a few important differences and
// additional requirements:
//
//  - They MUST monitor the g.waitFinished channel, and MUST finish
//    quickly after that channel is closed.
//  - They MUST not panic, as we don't bother to set up panic recovery
//    for them.
//  - The cfg.Workercontext() callback is not called.
//  - Returning 'nil' will not triggr a shutdown, even if
//    cfg.ShutdownOnNonError is set.
func (g *Group) goSupervisor(name string, fn func(ctx context.Context) error) {
	ctx := WithGoroutineName(g.baseCtx, ":"+name)
	g.goSupervisorCtx(ctx, fn)
}

// goSupervisorCtx() is like goSupervisor(), except it takes an
// already-created context.
func (g *Group) goSupervisorCtx(ctx context.Context, fn func(ctx context.Context) error) {
	g.supervisors.Add(1)
	go func() {
		var err error

		defer func() {
			if err != nil {
				g.goWorkerCtx(ctx, func(ctx context.Context) error {
					return err
				})
			}
			g.supervisors.Done()
		}()

		err = fn(ctx)
	}()
}

// Wait for all goroutines in the group to finish, and return returns
// an error if any of the workers errored or timed out.
//
// Once the group has initiated hard shutdown (either a 2nd shutdown
// signal was received, or the parent context is <-Done()), Wait will
// return within the HardShutdownTimeout passed to NewGroup.  If a
// poorly-behaved goroutine is still running at the end of that time,
// it is left running, and an error is returned.
func (g *Group) Wait() error {
	// 1. Wait for the worker goroutines to finish (or time out)
	shutdownCompleted := make(chan error)
	go func() {
		shutdownCompleted <- g.workers.Wait()
		close(shutdownCompleted)
	}()
	var ret error
	var timedOut bool
	select {
	case <-g.shutdownTimedOut:
		ret = errors.Errorf("failed to shut down within the %v shutdown timeout; some goroutines are left running", g.cfg.HardShutdownTimeout)
		timedOut = true
	case ret = <-shutdownCompleted:
	}

	// 2. Quit the supervisor goroutines
	close(g.waitFinished)
	g.supervisors.Wait()

	// 3. Belt-and-suspenders: Make sure that anything branched
	// from our Context observes that this group is no longer
	// running.
	g.hardCancel()

	// 4. Log the result and return
	if ret != nil && !g.cfg.DisableLogging {
		ctx := WithGoroutineName(g.baseCtx, ":shutdown_status")
		logGoroutineStatuses(ctx, "final goroutine statuses", dlog.Infof, g.List())
		if timedOut {
			logGoroutineTraces(ctx, "final goroutine stack traces", dlog.Errorf)
		}
	}
	return ret
}

// List wraps derrgroup.Group.List().
func (g *Group) List() map[string]derrgroup.GoroutineState {
	return g.workers.List()
}

type groupKey struct{}

// ParentGroup returns the Group that manages this goroutine/Context.
// If the Context is not managed by a Group, then nil is returned.
func ParentGroup(ctx context.Context) *Group {
	group := ctx.Value(groupKey{})
	if group == nil {
		return nil
	}
	return group.(*Group)
}
