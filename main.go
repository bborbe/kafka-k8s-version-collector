package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/bborbe/cron"
	flag "github.com/bborbe/flagenv"
	"github.com/bborbe/kafka-version-collector/version"
	"github.com/bborbe/run"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	defer glog.Flush()
	glog.CopyStandardLogTo("info")
	runtime.GOMAXPROCS(runtime.NumCPU())

	sender := &version.Sender{}
	fetcher := &version.Fetcher{
		HttpClient: http.DefaultClient,
	}
	syncer := version.Syncer{
		Fetcher: fetcher,
		Sender:  sender,
	}
	portPtr := flag.Int("port", 9002, "port to listen")
	waitPtr := flag.Duration("wait", time.Hour, "time to wait before next version collect")
	flag.StringVar(&sender.KafkaBrokers, "kafka-brokers", "", "kafka brokers")
	flag.StringVar(&sender.KafkaTopic, "kafka-topic", "", "kafka topic")
	flag.UintVar(&sender.KafkaSchemaId, "kafka-schema-id", 42, "kafka schema id")

	flag.Set("logtostderr", "true")
	flag.Parse()

	glog.V(0).Infof("Parameter Metrics-Port: %d", *portPtr)
	glog.V(0).Infof("Parameter Wait: %v", *waitPtr)
	glog.V(0).Infof("Parameter KafkaBrokers: %s", sender.KafkaBrokers)
	glog.V(0).Infof("Parameter KafkaTopic: %s", sender.KafkaTopic)

	if sender.KafkaBrokers == "" {
		glog.Exitf("KafkaBrokers missing")
	}
	if sender.KafkaTopic == "" {
		glog.Exitf("KafkaTopic missing")
	}
	if sender.KafkaSchemaId == 0 {
		glog.Exitf("KafkaSchemaId missing")
	}

	ctx := contextWithSig(context.Background())

	cronJob := cron.NewWaitCron(
		*waitPtr,
		syncer.Sync,
	)

	runServer := func(ctx context.Context) error {
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", *portPtr),
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

	glog.V(0).Infof("app started")
	if err := run.CancelOnFirstFinish(ctx, cronJob.Run, runServer); err != nil {
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
