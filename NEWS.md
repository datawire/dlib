# v1.2.5 (TBD)

 - Bugfix: `dlog`: v1.2.4 introduced a regression that broke existing
   external implementors of the `Logger` interface.  This has been
   fixed; implementations wishing to opt-in to v1.2.4's fast-logger
   behavior must now implement a distinct `OptimizedLogger` interface,
   which is not protected by the usual compatibility promises.

# v1.2.4 (2021-08-27)

 - Feature: `dlog`: Support deferring formatting of log messages to
   the `Logger` backends, so that if the log message would be dropped
   time isn't wasted formatting it just for the backend to drop it.
   This arguably should have triggered a v1.3.0 version bump.

 - Feature: `dcontext`: A new `WithoutCancel` function allows
   protecting an inner call from being canceled.  This arguably should
   have triggered a v1.3.0 version bump.

 - Feature: `derror`: `PanicToError`: Implement Go 1.13 error
   unwrapping.  This arguably should have triggered a v1.3.0 version
   bump.

 - Bugfix: `dexec`: Sort-of fix soft shutdown on `GOOS=windows`.  It
   is only possible to perform soft cancelation if
   `cmd.SysProcAttr.CreationFlags` includes
   `syscall.CREATE_NEW_PROCESS_GROUP`.  If `cmd.Start(ctx)` detects
   that the Context is soft and that bit isn't set, then it returns an
   error rather than starting the process.

 - Bugfix: `dcontext`: Fix a bug where
   `HardContext(WithoutContext(ctx))` can get canceled.

 - Minor: `dlog`: The default field order has changed.

 - Minor: `dexec`: Log when a signal is sent to the process.

 - Chore: Sync all borrowed files from the stdlib up to Go 1.15.14
   (from 1.15.5/1.15.6).

# v1.2.3 (2021-06-24)

 - Minor: `dexec`: The log formatting is now improved to take
   advantage of `dlog` functionality.

# v1.2.2 (2021-06-22)

 - Feature: `dlog`: A new `NewTestContextWithOpts` function allows
   greater configurability of the created logger.  This arguably
   should have triggered a v1.3.0 version bump.

# v1.2.1 (2021-03-08)

 - Bugfix: `dexec`: Fix a panic that occurs when the `Context` is
   canceled for a `Command` for which `.Start()` returned an error.

 - Minor: `dhttp`: Have better connection-worker goroutine names.

 - Chore: Our patches to `golang.org/x/net` have been merged upstream,
   so we have upgraded to that and no longer include a bundled copy of
   it that includes our patches.  This is not a user-facing change.

# v1.2.0 (2021-01-29)

 - Feature: Introduce the `dhttp` library.  The `dutil` HTTP functions
   are considered deprecated in favor of `dhttp`.

 - Change: Move `dutil.PanicToError` to `derror.PanicToError`, with a
   compatibility alias at `dutil.PanicToError`.

 - Minor: `dcontext`: `Context`s returned from `HardContext` now
   implement `fmt.Stringer` for better debugability.

 - Additionally, there are several news items regarding the
   now-deprecated `dutil` HTTP functions:

    + Feature: The HTTP functions now use `dlog` by default.
    + Bugfix: Correctly call `.Close()` on the underlying
      `net/http.Server` upon hard cancelation.
    + Bugfix: Document that it is an error to set `.BaseContext`, detect
      this error condition and return an error if it is encountered.
    + Bugfix: Be more careful about leaking resources

# v1.1.1 (2020-12-21)

 - Minor: `dgroup`: Be more intelligent about when to include or not
   include stacktraces with errors.

# v1.1.0 (2020-12-09)

 - Feature: Introduce the `dtime` library.

 - Change: Move `dutil.SleepWithContext` to `dtime.SleepWithContext`.
   This is a breaking change, but we allowed it anyway because it had
   only been around at `dutil` for 12 days.

# v1.0.0 (2020-12-01)

 - Feature: Initial public release.  This is mostly all fairly mature
   code being open-sourced from Ambassador Edge Stack.

 - Featur: Add `dutil.SleepWithContext` implementing cancelable
   sleep.
