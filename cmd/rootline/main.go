package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/pablontiv/picokit/autoupdate"
)

type updateClient interface {
	ApplyStagedIfAvailable() error
	FetchAndStage(currentVersion string) error
}

func newRootlineUpdater(ver string) *autoupdate.Updater {
	u := autoupdate.New("pablontiv/rootline", "rootline")
	u.CurrentVersion = ver
	u.VersionPolicy = autoupdate.SameMajorOnly
	return u
}

func runWithUpdater(ver string, execute func()) {
	runWithUpdaterUsing(ver, execute, newRootlineUpdater(ver), os.Stderr)
}

func runWithUpdaterUsing(ver string, execute func(), u updateClient, stderr io.Writer) {
	var noticeOnce sync.Once
	reportWithheld := func(err error) {
		var withheld *autoupdate.UpdateWithheldError
		if !errors.As(err, &withheld) {
			return
		}
		noticeOnce.Do(func() {
			_, _ = fmt.Fprintf(stderr, "rootline: incompatible update withheld: current %s, available %s; run the installer to upgrade deliberately\n", withheld.CurrentVersion, withheld.CandidateVersion)
		})
	}

	reportWithheld(u.ApplyStagedIfAvailable())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		reportWithheld(u.FetchAndStage(ver))
	}()

	execute()
	wg.Wait()
}

func main() {
	runWithUpdater(version, Execute)
}
