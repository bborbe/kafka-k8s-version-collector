// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/bborbe/kafka-k8s-version-collector/avro"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/seibert-media/go-kafka/schema"
)

//go:generate counterfeiter -o ../mocks/sender.go --fake-name Sender . Sender
type Sender interface {
	Send(ctx context.Context, versions <-chan avro.ApplicationVersionAvailable) error
}

func NewSender(
	producer sarama.SyncProducer,
	schemaRegistry schema.Registry,
	kafkaTopic string,
) Sender {
	return &sender{
		producer:       producer,
		schemaRegistry: schemaRegistry,
		kafkaTopic:     kafkaTopic,
	}
}

type sender struct {
	producer       sarama.SyncProducer
	schemaRegistry schema.Registry
	kafkaTopic     string
}

func (s *sender) Send(ctx context.Context, versions <-chan avro.ApplicationVersionAvailable) error {
	for {
		select {
		case <-ctx.Done():
			glog.V(3).Infof("context done => return")
			return nil
		case version, ok := <-versions:
			if !ok {
				glog.V(3).Infof("channel closed => return")
				return nil
			}
			schemaId, err := s.schemaRegistry.SchemaId(fmt.Sprintf("%s-value", s.kafkaTopic), version.Schema())
			if err != nil {
				return errors.Wrap(err, "get schema id failed")
			}
			buf := &bytes.Buffer{}
			if err := version.Serialize(buf); err != nil {
				return errors.Wrap(err, "serialize version failed")
			}
			partition, offset, err := s.producer.SendMessage(&sarama.ProducerMessage{
				Topic: s.kafkaTopic,
				Key:   sarama.StringEncoder(fmt.Sprintf("%s-%s", version.App, version.Version)),
				Value: &schema.AvroEncoder{SchemaId: schemaId, Content: buf.Bytes()},
			})
			if err != nil {
				return errors.Wrap(err, "send message to kafka failed")
			}
			glog.V(3).Infof("send message successful to %s with partition %d offset %d", s.kafkaTopic, partition, offset)
		}
	}
}
