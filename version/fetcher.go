// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bborbe/kafka-version-collector/avro"
	"github.com/pkg/errors"
)

type Fetcher struct {
	HttpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}
}

func (v *Fetcher) Fetch(ctx context.Context) ([]avro.Version, error) {
	url := "https://gcr.io/v2/google_containers/hyperkube-amd64/tags/list"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build request failed")
	}
	resp, err := v.HttpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request failed")
	}
	if resp.StatusCode/100 != 2 {
		return nil, errors.New("request status code != 2xx")
	}
	var data struct {
		Tags []string `json:"tags"`
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, errors.Wrap(err, "decode json failed")
	}
	var result []avro.Version
	for _, tag := range data.Tags {
		result = append(result, avro.Version{
			Number: tag,
			App:    "Kubernetes",
		})
	}
	return result, nil
}
