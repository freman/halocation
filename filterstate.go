package location

import (
	ghc "github.com/mkelcik/go-ha-client"
	"github.com/rs/zerolog/log"
)

// FilterState fulfills the State interface and filters inputs to other states
type FilterState struct {
	Parent State
}

// Set the entity to the state
func (filterState *FilterState) Set(value ghc.StateEntity) {
	_, hasLatitude := value.Attributes["latitude"].(float64)
	_, hasLongitude := value.Attributes["longitude"].(float64)

	if !(hasLatitude && hasLongitude) {
		log.Debug().Str("entity_id", value.EntityId).Msg("Missing coordinates")
		return
	}

	if position, has := value.Attributes["position"].(bool); has && !position {
		log.Debug().Str("entity_id", value.EntityId).Msg("Position is invalid")
		return
	}

	filterState.Parent.Set(value)
}

// List will call the parent state's List method
func (filterState *FilterState) List() ghc.StateEntities {
	return filterState.Parent.List()
}
