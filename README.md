# dlib - Small independent-but-complementary Context-oriented Go libraries

[![PkgGoDev](https://pkg.go.dev/badge/github.com/datawire/dlib)](https://pkg.go.dev/github.com/datawire/dlib)
[![Go Report Card](https://goreportcard.com/badge/github.com/datawire/dlib)](https://goreportcard.com/report/github.com/datawire/dlib)
[![CircleCI](https://circleci.com/gh/datawire/dlib.svg)](https://circleci.com/gh/datawire/dlib)
[![Coverage Status](https://coveralls.io/repos/github/datawire/dlib/badge.svg)](https://coveralls.io/github/datawire/dlib)

dlib is a set of small independent-but-complementary Context-oriented
Go libraries.

If each of the packages in dlib are independent, why are they lumped
together in to dlib?  They share common design principles:

 - Packages should be small.

   The way we use "small" is a little overloaded, and means a few
   different things:

    + Packages should be small in their functionality and API; the
      functionality should easily fit in the user's head.

	  A user should be able to quickly look at the package and
      understand the nugget of functionality that it provides.  A
      sprawling API is more things for the user to be distracted by
      and to try to fit in their head, taking space away from the
      actual problem they're trying to solve.

	  Don't make the user buy the whole enchilada if they just want
      the beans.

    + Packages should be small in their API.

	  The API of `dexec` is minimal; it's just that of stdlib
      `os/exec` that people already know (with one function removed,
      at that).  The size of the interface is mostly just typing
      `github.com/datawire/dlib/dexec` instead of `os/exec`.

	  Explicitly not part of the way we use "small" is "small in
      implementation".  Despite the small API of `dexec`, it has by
      far the most lines of code of any package in dlib.  This is
      because of the complexity in keeping the exact API of `os/exec`;
      this complexity is hidden by the simplicity of the user not
      having to learn something new.

	  Bigger APIs are more intimidating, harder to learn, harder to
      remember, and harder to discover.

    + Packages should be small in their opinions; the one opinion that
      they should cling to is "use Contexts!".

      The core of `dlog` doesn't actually do much of anything; it
      delegates to a pluggable logging backend.

      `dcontext` doesn't change the way you pass Contexts around, it
      doesn't force new opinions on code that interoperates with it; a
      special `dcontext` hard/soft Context can be passed to a
      non-`dcontext`-aware function, and the right thing will happen;
      a plain Context can be passed to a `dcontext`-aware function,
      and the right thing will happen.

   The way we use "small" is related to the way that [Rob Pike uses
   "simple"][Simplicity is Complicated video] ([slides][Simplicity is
   Complicated slides]).

   [Simplicity is Complicated video]: https://www.youtube.com/watch?v=rFejpH_tAHM
   [simplicity is Complicated slides]: https://talks.golang.org/2015/simplicity-is-complicated.slide

 - Packages should be independent.

   Similar to packages being small should be independent so as to not
   artificially increase their size.  If the user has to use both
   packages to use one, then are they really separate?  They're
   effectively one large package, but with worse discoverability.
   Packages being coupled means that you must now understand the
   functionality and API of both packages, and must accept the
   opinions of both packages.

   One package is free to use another internally, just as long as
   that's an implementation detail and not something that the user
   needs to care about.

 - Packages should be complementary.

   Despite being independent, the packages should complement each
   other.  You don't have to use `dcontext` if you're going to use
   `dexec`, but if you do, then you'll get graceful shutdown "for
   free".  You don't have to use `dlog` if you're going to use
   `dexec`, but if you do, then you'll be able to configure `dexec`'s
   output.

 - Packages should be Context-oriented.

   The one "opinion" that all of dlib clings to is to use Contexts.
   This allows us to reduce the other opinions that a package brings
   with it.

   Different logging solutions in Go are usually incompatible; do you
   pass around a `*log.Logger`, or a `logrus.FieldLogger`, or what;
   this opinion about logging affects all essentially all of your
   function signatures.  The opinion of "use Contexts" means: You're
   passing around a `context.Context` anyway, so let's attach the
   logger implementation to that, so that opinions about which logger
   has the prettiest don't need to affect the code that is written,
   except for one-time setup in the final application's `main()`.

 - Defaults should be useful.

   The core of `dlog` doesn't actually do much of anything; it
   delegates to a pluggable logging backend, but it uses a
   `logrus`-based backend by default; few users will be upset by this
   default logging with colorized output and timestamps.  Having
   useful defaults is a backing-assumption for being Context-oriented.
   If `dlog` didn't have a useful default logger, then using it
   wouldn't be a no-brainer, using `dlog` would force the user of that
   package to care about `dlog` and whether or not they'd taken care
   to configure the logger ahead-of-time.

   A zero `dgroup.GroupConfig{}` is useful without filling in any
   settings; things that are on by default have a `DisableXXX` bool,
   and things that are off by default have an `EnableXXX` bool.  The
   most-common configuration will be empty, and the second-most-common
   configuration will be the just the 1 item `EnableSignalHandling:
   true` (which we can't make the default because it would be bad to
   set it up multiple signal handler in the same program).

In all, dlib is lumped together so that the user can trust that these
principles have been reasonably consistently applied.  The user can
pull in one package from dlib and trust that they won't have to worry
about having to adjust their program to that package's opinions
(except of course, for the opinion that you should use Contexts!).
