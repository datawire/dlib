package dcontext_test

import (
	"context"
	"fmt"
	"time"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dhttp"
)

// This should be a very simple example of a parent caller function, showing how
// to manage a hard/soft Context and how to call code that is dcontext-aware.
func ExampleWithSoftness() {
	ctx := context.Background()               // Context is hard by default
	ctx, timeToDie := context.WithCancel(ctx) // hard Context => hard cancel
	defer timeToDie()
	ctx = dcontext.WithSoftness(ctx)                  // make it soft
	ctx, startShuttingDown := context.WithCancel(ctx) // soft Context => soft cancel

	retCh := make(chan error)
	go func() {
		sc := &dhttp.ServerConfig{
			// ...
		}
		retCh <- sc.ListenAndServe(ctx, ":0")
	}()

	// Run for a while.
	time.Sleep(3 * time.Second)

	// Shut down.
	startShuttingDown() // Soft shutdown; start draining connections.
	select {
	case err := <-retCh:
		// It shut down fine with just the soft shutdown; everything was
		// well-behaved.  It isn't necessary to cut shutdown short by
		// triggering a hard shutdown with timeToDie() in this case.
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("soft shutdown")
		}
	case <-time.After(2 * time.Second): // shutdown grace period
		// It's taking too long to shut down--it seems that some clients
		// are refusing to hang up.  So now we trigger a hard shutdown
		// and forcefully close the connections.  This will cause errors
		// for those clients.
		timeToDie() // Hard shutdown; cause errors for clients
		if err := <-retCh; err != nil {
			fmt.Println(err.Error())
		}
	}
	// Output:
	// soft shutdown
}
