// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version_test

import (
	"context"
	"errors"

	mocksmocks "github.com/Shopify/sarama/mocks"
	"github.com/bborbe/kafka-k8s-version-collector/avro"
	"github.com/bborbe/kafka-k8s-version-collector/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/seibert-media/go-kafka/mocks"
)

var _ = Describe("Version Sender", func() {
	var sender version.Sender
	var producer *mocksmocks.SyncProducer
	var topic string
	var schemaRegistry *mocks.SchemaRegistry
	BeforeEach(func() {
		var t GinkgoTestReporter
		producer = mocksmocks.NewSyncProducer(t, nil)
		topic = "my-topic"
		schemaRegistry = &mocks.SchemaRegistry{}
		sender = version.NewSender(
			producer,
			schemaRegistry,
			topic,
		)
	})
	It("send until channel is closed", func() {
		versions := make(chan avro.ApplicationVersionAvailable)
		close(versions)
		err := sender.Send(context.Background(), versions)
		Expect(err).NotTo(HaveOccurred())
	})
	It("send until channel is closed", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		versions := make(chan avro.ApplicationVersionAvailable)
		defer close(versions)
		err := sender.Send(ctx, versions)
		Expect(err).NotTo(HaveOccurred())
	})
	It("send version to producer", func() {
		counter := 0
		producer.ExpectSendMessageWithCheckerFunctionAndSucceed(func(val []byte) error {
			counter++
			return nil
		})
		versions := make(chan avro.ApplicationVersionAvailable, 2)
		versions <- *avro.NewApplicationVersionAvailable()
		close(versions)
		err := sender.Send(context.Background(), versions)
		Expect(err).To(BeNil())
		Expect(counter).To(Equal(1))
	})
	It("returns error if get schemaId fails", func() {
		schemaRegistry.SchemaIdReturns(0, errors.New("banana"))
		versions := make(chan avro.ApplicationVersionAvailable, 2)
		versions <- *avro.NewApplicationVersionAvailable()
		close(versions)
		err := sender.Send(context.Background(), versions)
		Expect(err).To(HaveOccurred())
	})
	It("returns error if send message fails", func() {
		producer.ExpectSendMessageAndFail(errors.New("banana"))
		versions := make(chan avro.ApplicationVersionAvailable, 2)
		versions <- *avro.NewApplicationVersionAvailable()
		close(versions)
		err := sender.Send(context.Background(), versions)
		Expect(err).To(HaveOccurred())
	})
})
