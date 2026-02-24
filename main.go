// Package main is the entry point for the anisan-cli application.
package main

import (
	"github.com/anisan-cli/anisan/cmd"
	"github.com/anisan-cli/anisan/config"
	"github.com/anisan-cli/anisan/internal/cache"
	"github.com/anisan-cli/anisan/internal/sync"
	"github.com/anisan-cli/anisan/log"
	"github.com/samber/lo"
)

func main() {
	lo.Must0(config.Setup())
	lo.Must0(log.Setup())

	// Initialize asynchronous background processes for cache maintenance and synchronization.
	go cache.CollectGarbage()
	go sync.ReconcileFailures()

	cmd.Execute()
}
