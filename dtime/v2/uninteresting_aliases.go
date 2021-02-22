// This file deals with uninteresting boilerplate related to making dtime be closer to a drop-in
// replacement for stdlib time.

package dtime

import (
	"context"
	"time"
)

// Miscellaneous ///////////////////////////////////////////////////////////////////////////////////

// These are predefined layouts for use in Time.Format and dtime.Parse.  See the documentation in
// stdlib time for more information.
const (
	ANSIC       = time.ANSIC
	UnixDate    = time.UnixDate
	RubyDate    = time.RubyDate
	RFC822      = time.RFC822
	RFC822Z     = time.RFC822Z
	RFC850      = time.RFC850
	RFC1123     = time.RFC1123
	RFC1123Z    = time.RFC1123Z
	RFC3339     = time.RFC3339
	RFC3339Nano = time.RFC3339Nano
	Kitchen     = time.Kitchen
	Stamp       = time.Stamp
	StampMilli  = time.StampMilli
	StampMicro  = time.StampMicro
	StampNano   = time.StampNano
)

// ParseError describes a problem parsing a time string.  See the documentation in stdlib time for
// more information.
type ParseError = time.ParseError

// Calendar ////////////////////////////////////////////////////////////////////////////////////////

// A Month specifies a month of the year (January = 1, ...).  See the documentation in stdlib time
// for more information.
type Month = time.Month

const (
	January   = time.January
	February  = time.February
	March     = time.March
	April     = time.April
	May       = time.May
	June      = time.June
	July      = time.July
	August    = time.August
	September = time.September
	October   = time.October
	November  = time.November
	December  = time.December
)

// A Weekday specifies a day of the week (Sunday = 0, ...).  See the documentation in stdlib time
// for more information.
type Weekday = time.Weekday

const (
	Sunday    = time.Sunday
	Monday    = time.Monday
	Tuesday   = time.Tuesday
	Wednesday = time.Wednesday
	Thursday  = time.Thursday
	Friday    = time.Friday
	Saturday  = time.Saturday
)

// Convenience functions ///////////////////////////////////////////////////////////////////////////

// After either waits for the Duration to elapse and then sends the current time on the returned
// channel, or waits for the Context to become Done and closes the returned channel; whichever
// happens first.  If the Duration elapses before the Context becomes Done, then the returned
// channel will never be closed.
//
// It is equivalent to `NewTimer(ctx, d).C`.
//
// The underlying Timer is not recovered by the garbage collector until either the timer fires or
// the Context becomes Done.  If efficiency is a concern, be sure to cancel the Context when the
// timer is no longer needed.
func After(ctx context.Context, d Duration) <-chan Time {
	return NewTimer(ctx, d).C
}

// Sleep pauses the current goroutine either for at least the Duration d, or until the Context
// becomes done; whichever occurs first.  A negative or zero duration causes Sleep to return
// immediately.
//
// It is equivalent to `<-After(ctx, d)`.
func Sleep(ctx context.Context, d Duration) {
	<-After(ctx, d)
}

// Tick is a convenience wrapper for NewTicker providing access to the ticking channel only.  While
// Unlike NewTicker, Tick will return nil if d <= 0; otherwise it is equivalent to `NewTicker(ctx,
// d).C`.
func Tick(ctx context.Context, d Duration) <-chan Time {
	if d < 0 {
		return nil
	}
	return NewTicker(ctx, d).C
}

// type: Duration //////////////////////////////////////////////////////////////////////////////////

// A Duration represents the elapsed time between two instants as an int64 nanosecond count.  See
// the documentation in stdlib time for more information.
type Duration = time.Duration

// Common durations.  There is no definition for units of Day or larger to avoid confusion across
// daylight savings time zone transitions.  See the documentation in stdlib time for more
// information.
const (
	Nanosecond  = time.Nanosecond
	Microsecond = time.Microsecond
	Millisecond = time.Millisecond
	Second      = time.Second
	Minute      = time.Minute
	Hour        = time.Hour
)

// ParseDuration parses a duration string.  It is an alias for stdlib time.ParseDuration; see the
// documentation there for more information.
func ParseDuration(s string) (Duration, error) {
	return time.ParseDuration(s)
}

// Since returns the time elapsed since t.  It is shorthand for `dtime.Now(ctx).Sub(t)`.
func Since(ctx context.Context, t Time) Duration {
	return Now(ctx).Sub(t)
}

// Until returns the duration until t.  It is shorthand for `t.Sub(dtime.Now(ctx))`.
func Until(ctx context.Context, t Time) Duration {
	return t.Sub(Now(ctx))
}

// type: Location //////////////////////////////////////////////////////////////////////////////////

// A Location maps time instants to the zone in use at that time.  See the documentation in stdlib
// time for more information.
type Location = time.Location

// Local returns the system's local time zone.
//
// BUG(lukeshu): It is not possible to spoof the system's local timezone.  It would be a good
// feature to have, but making it possible would require wrapping (rather than aliasing) the
// `time.Time` type (in order to change the `.Local()` and `.UnmarshalBinary()` methods), which we
// view to be too great a cost.
func Local() *Location {
	// This is a function instead of a variable so that no one gets a hair-brained idea that
	// they can set it (we can't just declare it as `const` because you can't have a const
	// pointer).
	return time.Local
}

// UTC returns the Location representing Universal Coordinated Time (UTC).
func UTC() *Location {
	// This is a function instead of a variable so that no one gets a hair-brained idea that
	// they can set it (we can't just declare it as `const` because you can't have a const
	// pointer).
	return time.UTC
}

// LoadLocation returns the Location with the given name.  See the documentation in stdlib time for
// more information.
func LoadLocation(name string) (*Location, error) {
	return time.LoadLocation(name)
}

// LoadLocationFromTZData returns a Location with the given name initialized from the IANA Time Zone
// database-formatted data.  See the documentation in stdlib time for more information.
func LoadLocationFromTZData(name string, data []byte) (*Location, error) {
	return time.LoadLocationFromTZData(name, data)
}

// type: Time //////////////////////////////////////////////////////////////////////////////////////

// A Time represents an instant in time with nanosecond precision.  See the documentation in stdlib
// time for more information.
type Time = time.Time

// Date returns the Time corresponding to
//
//     yyyy-mm-dd hh:mm:ss + nsec nanoseconds
//
// in the appropriate zone for that time in the given location.
//
// See the documentation in stdlib time for more information.
func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time {
	return time.Date(year, month, day, hour, min, sec, nsec, loc)
}

// Parse parses a formatted string and returns the time value it represents.  See the documentation
// in stdlib time for more information.
func Parse(layout, value string) (Time, error) {
	return time.Parse(layout, value)
}

// ParseInLocation is like Parse but differs in two important ways.  First, in the absence of time
// zone information, Parse interprets a time as UTC; ParseInLocation interprets the time as in the
// given location.  Second, when given a zone offset or abbreviation, Parse tries to match it
// against the Local location; ParseInLocation uses the given location.
func ParseInLocation(layout, value string, loc *Location) (Time, error) {
	return time.ParseInLocation(layout, value, loc)
}

// Unix returns the local Time corresponding to the given Unix time, sec seconds and nsec
// nanoseconds since January 1, 1970 UTC.  See the documentation in stdlib time for more
// information.
func Unix(sec int64, nsec int64) Time {
	return time.Unix(sec, nsec)
}
