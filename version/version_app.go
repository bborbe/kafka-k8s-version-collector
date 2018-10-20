package version

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bborbe/cron"
	"github.com/bborbe/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/seibert-media/go-kafka/schema"
)

type App struct {
	Wait              time.Duration
	Port              int
	KafkaBrokers      string
	KafkaTopic        string
	SchemaRegistryUrl string
}

func (a *App) Validate() error {
	if a.Port <= 0 {
		return errors.New("Port missing")
	}
	if a.Wait <= 0 {
		return errors.New("Wait missing")
	}
	if a.KafkaBrokers == "" {
		return errors.New("KafkaBrokers missing")
	}
	if a.KafkaTopic == "" {
		return errors.New("KafkaTopic missing")
	}
	if a.SchemaRegistryUrl == "" {
		return errors.New("SchemaRegistryUrl missing")
	}
	return nil
}

func (a *App) Run(ctx context.Context) error {
	return run.CancelOnFirstFinish(
		ctx,
		a.runCron,
		a.runHttpServer,
	)
}

func (a *App) runCron(ctx context.Context) error {
	syncer := Syncer{
		Fetcher: &Fetcher{
			HttpClient: http.DefaultClient,
		},
		Sender: &Sender{
			KafkaTopic:   a.KafkaTopic,
			KafkaBrokers: a.KafkaBrokers,
			SchemaRegistry: &schema.Registry{
				HttpClient:        http.DefaultClient,
				SchemaRegistryUrl: a.SchemaRegistryUrl,
			},
		},
	}

	cronJob := cron.NewWaitCron(
		a.Wait,
		syncer.Sync,
	)
	return cronJob.Run(ctx)
}

func (a *App) runHttpServer(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.Port),
		Handler: promhttp.Handler(),
	}
	go func() {
		select {
		case <-ctx.Done():
			server.Shutdown(ctx)
		}
	}()
	return server.ListenAndServe()
}
