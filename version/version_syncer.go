// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"context"
	"runtime"

	"github.com/bborbe/kafka-k8s-version-collector/avro"
	"github.com/bborbe/run"
	"github.com/golang/glog"
)

//go:generate counterfeiter -o ../mocks/syncer.go --fake-name Syncer . Syncer
type Syncer interface {
	Sync(ctx context.Context) error
}

func NewSyncer(
	fetcher Fetcher,
	sender Sender,
) Syncer {
	return &syncer{
		fetcher: fetcher,
		sender:  sender,
	}
}

type syncer struct {
	fetcher Fetcher
	sender  Sender
}

func (s *syncer) Sync(ctx context.Context) error {
	glog.V(1).Infof("sync started")
	defer glog.V(1).Infof("sync finished")
	versions := make(chan avro.ApplicationVersionAvailable, runtime.NumCPU())
	return run.CancelOnFirstError(
		ctx,
		func(ctx context.Context) error {
			defer close(versions)
			return s.fetcher.Fetch(ctx, versions)
		},
		func(ctx context.Context) error {
			return s.sender.Send(ctx, versions)
		},
	)
}
