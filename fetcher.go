package location

import (
	"context"
	"sync"
	"time"

	ghc "github.com/mkelcik/go-ha-client"
	"github.com/rs/zerolog/log"
)

// Fetcher is a simple polling tool to poll for states in Home Assistant
type Fetcher struct {
	MaxConcurrency int
	PollInterval   time.Duration
	State          State
	Client         interface {
		GetStateForEntity(ctx context.Context, entityId string) (ghc.StateEntity, error)
	}
	Entities []string
}

// Run the fetcher in a loop
func (fetcher *Fetcher) Run(ctx context.Context) {
	ticker := time.NewTicker(fetcher.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetcher.fetchEntities(ctx)
		}
	}
}

func (fetcher *Fetcher) fetchEntities(ctx context.Context) {
	waitCh := make(chan struct{}, fetcher.MaxConcurrency)
	var wg sync.WaitGroup

	wg.Add(len(fetcher.Entities))
	for _, entity := range fetcher.Entities {
		waitCh <- struct{}{}
		go func(entity string) {
			defer func() {
				<-waitCh
				wg.Done()
			}()

			fetcher.fetchEntity(ctx, entity)
		}(entity)
	}

	wg.Wait()
	close(waitCh)
}

func (fetcher *Fetcher) fetchEntity(ctx context.Context, entity string) {
	estate, err := fetcher.Client.GetStateForEntity(ctx, entity)
	if err != nil {
		log.Warn().Err(err).Str("entity_id", entity).Msg("Failed to get state.")
		return
	}

	if position, has := estate.Attributes["position"].(bool); has && !position {
		log.Debug().Str("entity_id", entity).Msg("Position is invalid")
		return
	}

	fetcher.State.Set(estate)
}
