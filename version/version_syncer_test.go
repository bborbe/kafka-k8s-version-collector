// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version_test

import (
	"context"

	"github.com/bborbe/kafka-k8s-version-collector/avro"
	"github.com/bborbe/kafka-k8s-version-collector/mocks"
	"github.com/bborbe/kafka-k8s-version-collector/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version Sender", func() {
	var syncer version.Syncer
	var sender *mocks.Sender
	var fetcher *mocks.Fetcher
	var sendCounter int
	BeforeEach(func() {
		sendCounter = 0
		sender = &mocks.Sender{}
		sender.SendStub = func(ctx context.Context, availables <-chan avro.ApplicationVersionAvailable) error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case _, ok := <-availables:
					if !ok {
						return nil
					}
					sendCounter++
				}
			}
		}
		fetcher = &mocks.Fetcher{}
		syncer = version.NewSyncer(
			fetcher,
			sender,
		)
	})
	It("returns without error", func() {
		err := syncer.Sync(context.Background())
		Expect(err).NotTo(HaveOccurred())
	})
	It("alls send", func() {
		fetcher.FetchStub = func(ctx context.Context, availables chan<- avro.ApplicationVersionAvailable) error {
			select {
			case <-ctx.Done():
				return nil
			case availables <- *avro.NewApplicationVersionAvailable():
				return nil
			}
		}
		err := syncer.Sync(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(sendCounter).To(Equal(1))
	})
	It("sends all fetched versions", func() {
		counter := 10
		fetcher.FetchStub = func(ctx context.Context, availables chan<- avro.ApplicationVersionAvailable) error {
			for i := 0; i < counter; i++ {
				select {
				case <-ctx.Done():
					return nil
				case availables <- *avro.NewApplicationVersionAvailable():
				}
			}
			return nil
		}
		err := syncer.Sync(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(sendCounter).To(Equal(counter))
	})
})
