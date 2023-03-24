package location

import (
	"sync"

	ghc "github.com/mkelcik/go-ha-client"
)

// State is an interface for storing state
type State interface {
	Set(value ghc.StateEntity)
	List() ghc.StateEntities
}

// LastState fulfills the State interface and stores the last known state
type LastState struct {
	mu    sync.RWMutex
	state map[string]ghc.StateEntity
}

// Set the entity to the state
func (lastState *LastState) Set(value ghc.StateEntity) {
	lastState.mu.Lock()
	defer lastState.mu.Unlock()

	if lastState.state == nil {
		lastState.state = make(map[string]ghc.StateEntity)
	}

	lastState.state[value.EntityId] = value
}

// List all current entities
func (lastState *LastState) List() (result ghc.StateEntities) {
	for _, value := range lastState.state {
		result = append(result, value)
	}

	return result
}

// EmittingState will fire a callback when the state is set
type EmittingState struct {
	OnState func(value ghc.StateEntity)
	State
}

// Set the entity to the state
func (emittingState *EmittingState) Set(value ghc.StateEntity) {
	if emittingState.State != nil {
		emittingState.State.Set(value)
	}

	if emittingState.OnState != nil {
		emittingState.OnState(value)
	}
}
