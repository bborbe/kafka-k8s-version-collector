package version

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/bborbe/kafka-version-collector/avro"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type Sender struct {
	KafkaBrokers   string
	KafkaTopic     string
	SchemaRegistry interface {
		SchemaId(subject string, object interface {
			Schema() string
		}) (uint32, error)
	}
}

func (s *Sender) Send(ctx context.Context, versions []avro.Version) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_0_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 10
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(strings.Split(s.KafkaBrokers, ","), config)
	if err != nil {
		return errors.Wrap(err, "create sync producer failed")
	}
	defer producer.Close()

	for _, version := range versions {
		schemaId, err := s.SchemaRegistry.SchemaId(fmt.Sprintf("%s-value", s.KafkaTopic), &version)
		if err != nil {
			return errors.Wrap(err, "get schema id failed")
		}
		bs := make([]byte, 4)
		binary.BigEndian.PutUint32(bs, schemaId)
		b := bytes.NewBuffer(append([]byte{0}, bs...))
		if err := version.Serialize(b); err != nil {
			return errors.Wrap(err, "serialize version failed")
		}
		partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: s.KafkaTopic,
			Key:   sarama.StringEncoder(fmt.Sprintf("%s-%s", version.App, version.Number)),
			Value: sarama.ByteEncoder(b.Bytes()),
		})
		if err != nil {
			return errors.Wrap(err, "send message to kafka failed")
		}
		glog.V(0).Infof("send message successful to %s with partition %d offset %d", s.KafkaTopic, partition, offset)
	}
	return nil
}
