package main

import (
	"sync"

	"github.com/pablontiv/picokit/autoupdate"
)

func runWithUpdater(ver string, execute func()) {
	u := autoupdate.New("pablontiv/rootline", "rootline")
	u.CurrentVersion = ver
	_ = u.ApplyStagedIfAvailable()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = u.FetchAndStage(ver)
	}()

	execute()
	wg.Wait()
}

func main() {
	runWithUpdater(version, Execute)
}
