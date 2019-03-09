// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The kafka-version-collector collects available versions of software and publish it to a topic.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/bborbe/argument"
	"github.com/bborbe/cron"
	flag "github.com/bborbe/flagenv"
	"github.com/bborbe/kafka-k8s-version-collector/version"
	"github.com/bborbe/run"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/seibert-media/go-kafka/schema"
)

func main() {
	defer glog.Flush()
	glog.CopyStandardLogTo("info")
	runtime.GOMAXPROCS(runtime.NumCPU())
	_ = flag.Set("logtostderr", "true")

	app := &application{}
	if err := argument.Parse(app); err != nil {
		glog.Exitf("parse app failed: %v", err)
	}

	glog.V(0).Infof("app started")
	if err := app.Run(contextWithSig(context.Background())); err != nil {
		glog.Exitf("app failed: %+v", err)
	}
	glog.V(0).Infof("app finished")
}

func contextWithSig(ctx context.Context) context.Context {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-signalCh:
		case <-ctx.Done():
		}
	}()

	return ctxWithCancel
}

type application struct {
	Wait              time.Duration `required:"true" arg:"wait" env:"WAIT" default:"1h" usage:"time to wait before next version collect"`
	Port              int           `required:"true" arg:"port" env:"PORT" default:"9003" usage:"port to listen"`
	KafkaBrokers      string        `required:"true" arg:"kafka-brokers" env:"KAFKA_BROKERS" usage:"kafka brokers"`
	KafkaTopic        string        `required:"true" arg:"kafka-topic" env:"KAFKA_TOPIC" usage:"kafka topic"`
	SchemaRegistryUrl string        `required:"true" arg:"kafka-schema-registry-url" env:"KAFKA_SCHEMA_REGISTRY_URL" usage:"kafka schema registry url"`
}

func (a *application) Run(ctx context.Context) error {
	return run.CancelOnFirstFinish(
		ctx,
		a.runCron,
		a.runHttpServer,
	)
}

func (a *application) runCron(ctx context.Context) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_0_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 10
	config.Producer.Return.Successes = true

	client, err := sarama.NewClient(strings.Split(a.KafkaBrokers, ","), config)
	if err != nil {
		return errors.Wrap(err, "create client failed")
	}
	defer client.Close()

	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return errors.Wrap(err, "create sync producer failed")
	}
	defer producer.Close()

	httpClient := http.DefaultClient
	syncer := version.NewSyncer(
		version.NewFetcher(httpClient, "https://gcr.io"),
		version.NewSender(
			producer,
			schema.NewRegistry(
				httpClient,
				a.SchemaRegistryUrl,
			),
			a.KafkaTopic,
		),
	)

	cronJob := cron.NewWaitCron(
		a.Wait,
		syncer.Sync,
	)
	return cronJob.Run(ctx)
}

func (a *application) runHttpServer(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.Port),
		Handler: promhttp.Handler(),
	}
	go func() {
		select {
		case <-ctx.Done():
			if err := server.Shutdown(ctx); err != nil {
				glog.Warningf("shutdown failed: %v", err)
			}
		}
	}()
	err := server.ListenAndServe()
	if err == http.ErrServerClosed {
		glog.V(0).Info(err)
		return nil
	}
	return errors.Wrap(err, "httpServer failed")
}
