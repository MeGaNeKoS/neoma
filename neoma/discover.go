package neoma

import (
	"sync"

	"github.com/MeGaNeKoS/neoma/core"
)

var (
	discoveredMu     sync.RWMutex
	discoveredErrors = map[string][]core.DiscoveredError{}
)

func RegisterDiscoveredErrors(errors map[string][]core.DiscoveredError) {
	discoveredMu.Lock()
	defer discoveredMu.Unlock()
	for k, v := range errors {
		discoveredErrors[k] = v
	}
}

func getDiscoveredErrors(handlerName string) []core.DiscoveredError {
	discoveredMu.RLock()
	defer discoveredMu.RUnlock()
	return discoveredErrors[handlerName]
}
