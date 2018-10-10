package version

import (
	"context"

	"github.com/bborbe/kafka-version-collector/avro"
	"github.com/pkg/errors"
)

type Syncer struct {
	Fetcher interface {
		Fetch(ctx context.Context) ([]avro.Version, error)
	}
	Sender interface {
		Send(ctx context.Context, versions []avro.Version) error
	}
}

func (s *Syncer) Sync(ctx context.Context) error {
	versions, err := s.Fetcher.Fetch(ctx)
	if err != nil {
		return errors.Wrap(err, "fetch versions failed")
	}
	return errors.Wrap(s.Sender.Send(ctx, versions), "send version failed")
}
