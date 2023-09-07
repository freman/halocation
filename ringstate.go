package location

import (
	"container/ring"
	"sync"

	ghc "github.com/mkelcik/go-ha-client"
)

// RingState fulfills the State interface and stores a ring buffer of the last known state
type RingState struct {
	mu    sync.RWMutex
	state map[string]*ring.Ring
	Size  int
}

// Set the entity to the state
func (ringState *RingState) Set(value ghc.StateEntity) {
	ringState.mu.Lock()
	defer ringState.mu.Unlock()

	if ringState.state == nil {
		ringState.state = make(map[string]*ring.Ring)
	}

	if _, has := ringState.state[value.EntityId]; !has {
		ringState.state[value.EntityId] = ring.New(ringState.Size)
	}

	ringState.state[value.EntityId].Value = value
	ringState.state[value.EntityId] = ringState.state[value.EntityId].Next()
}

// List all current entities from their rings
func (ringState *RingState) List() (result ghc.StateEntities) {
	ringState.mu.Lock()
	defer ringState.mu.Unlock()

	for _, value := range ringState.state {
		value.Do(func(v interface{}) {
			if ghcv, isa := v.(ghc.StateEntity); isa {
				result = append(result, ghcv)
			}
		})
	}

	return result
}
