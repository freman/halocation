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
	Client         *ghc.Client
	Entities       []string
	Bootstrap      time.Duration
}

type fetchFunc func(ctx context.Context, entity string)

// Run the fetcher in a loop
func (fetcher *Fetcher) Run(ctx context.Context) {
	log.Debug().Strs("entities", fetcher.Entities).Msg("polling for entitites")

	ticker := time.NewTicker(fetcher.PollInterval)
	defer ticker.Stop()

	if fetcher.Bootstrap > 0 {
		log.Debug().Dur("duration", fetcher.Bootstrap).Msg("Bootstrapping entities.")
		fetcher.fetchEntityHistories(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetcher.fetchEntities(ctx)
		}
	}
}

func (fetcher *Fetcher) fetchConcurrent(ctx context.Context, fn fetchFunc) {
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

			fn(ctx, entity)
		}(entity)
	}

	wg.Wait()
	close(waitCh)
}

func (fetcher *Fetcher) fetchEntities(ctx context.Context) {
	fetcher.fetchConcurrent(ctx, fetcher.fetchEntity)
}

func (fetcher *Fetcher) fetchEntityHistories(ctx context.Context) {
	fetcher.fetchConcurrent(ctx, fetcher.fetchEntityHistory)
}

func (fetcher *Fetcher) fetchEntity(ctx context.Context, entity string) {
	estate, err := fetcher.Client.GetStateForEntity(ctx, entity)
	if err != nil {
		log.Warn().Err(err).Str("entity_id", entity).Msg("Failed to get state.")
		return
	}

	fetcher.State.Set(estate)
}

func (fetcher *Fetcher) fetchEntityHistory(ctx context.Context, entity string) {
	estates, err := fetcher.Client.GetStateChangesHistory(ctx, &ghc.StateChangesFilter{
		StartTime:              time.Now().Add(-1 * fetcher.Bootstrap),
		EndTime:                time.Now(),
		FilterEntityId:         entity,
		SignificantChangesOnly: true,
	})

	if err != nil {
		log.Warn().Err(err).Str("entity_id", entity).Msg("Failed to get state change history.")
		return
	}

	// We're not using a wildcard, there should only be one set of changes returned
	if len(estates) != 1 {
		log.Warn().Err(err).Str("entity_id", entity).Msg("Unexpected state change history returned.")
		return
	}

	for i := range estates[0] {
		// Have to copy data here :( - thankfully it's only bootstrap ;)
		fetcher.State.Set(ghc.StateEntity{
			EntityId:    estates[0][i].EntityId,
			State:       estates[0][i].State,
			Attributes:  estates[0][i].Attributes,
			LastChanged: estates[0][i].LastChanged,
			LastUpdated: estates[0][i].LastUpdated,
		})
	}
}
