package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/freman/sse"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	ghc "github.com/mkelcik/go-ha-client"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	location "github.com/freman/halocation"
)

func main() {
	var entities arrayFlags

	haToken := flag.String("token", os.Getenv("HA_TOKEN"), "Home Assistant token [env: HA_TOKEN]")
	haURL := flag.String("url", os.Getenv("HA_URL"), "Home Assistant URL [env: HA_URL]")
	pollInterval := flag.Duration("poll-interval", 5*time.Second, "Rate of polling")
	maxConcurrency := flag.Int("concurrency", 2, "Polling concurrency")
	listen := flag.String("listen", ":9922", "Listen configuration for HTTP traffic")
	logLevel := flag.String("log-level", zerolog.LevelInfoValue, "Log level")
	flag.Var(&entities, "entity", "Entity ID to export, repeat flag or comma separate for more")
	help := flag.Bool("help", false, "Show command arguments")

	flag.Parse()

	if *help {
		flag.PrintDefaults()
		return
	}
	setupLogging(*logLevel)

	log.Debug().Strs("entities", entities).Msg("polling for entitites")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := ghc.NewClient(ghc.ClientConfig{Token: *haToken, Host: *haURL}, &http.Client{
		Timeout: 30 * time.Second,
	})

	if err := initialPing(ctx, client); err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed to ping Home Assistant.")
	}

	state := &location.EmittingState{
		State: &location.LastState{},
	}

	fetcher := location.Fetcher{
		MaxConcurrency: *maxConcurrency,
		PollInterval:   *pollInterval,
		State:          state,
		Entities:       []string(entities),
		Client:         client,
	}

	bus := sse.New(sse.WithOnConnect(func(e sse.Emitter) {
		list := state.List()
		log.Debug().Int("states", len(list)).Msg("onConnect - dumping states")
		for i := range list {
			e.Emit(list[i])
		}
	}))

	state.OnState = func(value ghc.StateEntity) {
		log.Trace().Str("entity_id", value.EntityId).Msg("Emitting state")
		bus.Emit(value)
	}

	go fetcher.Run(ctx)

	e := setupEcho(bus, client)

	go handleSignals(ctx, e)

	if err := e.Start(*listen); err != nil {
		log.Fatal().Err(err).Msg("Failed to start listener")
	}
}

func setupLogging(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	lvl, err := zerolog.ParseLevel(level)

	if err == nil {
		zerolog.SetGlobalLevel(lvl)
		return
	}

	log.Fatal().Err(err).Msg("Failed to configure log level")
}

func setupEcho(bus *sse.EventStream, client *ghc.Client) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info().
				Str("URI", v.URI).
				Int("status", v.Status).
				Msg("request")

			return nil
		},
	}))

	e.GET("/sse", echo.WrapHandler(bus))
	e.GET("/health", func(ec echo.Context) error {
		if err := client.Ping(ec.Request().Context()); err != nil {
			return ec.Blob(http.StatusFailedDependency, "text/plain; charset=UTF-8", []byte(err.Error()))
		}

		return ec.Blob(http.StatusOK, "text/plain; charset=UTF-8", []byte("ok"))
	})

	return e
}

func handleSignals(ctx context.Context, server interface{ Shutdown(context.Context) error }) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigint

	log.Info().Str("signal", sig.String()).Msg("Signal received")

	ctx, cancelFunc := context.WithTimeout(ctx, time.Minute)
	defer cancelFunc()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	}

	log.Info().Msg("Application shutdown")
}

func initialPing(ctx context.Context, pinger interface{ Ping(context.Context) error }) error {
	backoffMethod := backoff.WithContext(
		backoff.WithMaxRetries(
			backoff.NewExponentialBackOff(),
			3,
		),
		ctx,
	)

	return backoff.Retry(func() error {
		log.Trace().Msg("Attempting to ping your Home Assistant")
		return pinger.Ping(ctx)
	}, backoffMethod)
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	if strings.Contains(value, ",") {
		for _, v := range strings.Split(value, ",") {
			*i = append(*i, strings.TrimSpace(v))
		}

		return nil
	}

	*i = append(*i, value)

	return nil
}
