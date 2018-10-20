// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/bborbe/kafka-k8s-version-collector/avro"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o ../mocks/http_client.go --fake-name HttpClient . httpClient
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Fetcher struct {
	HttpClient httpClient
}

func (v *Fetcher) Fetch(ctx context.Context, versions chan<- avro.ApplicationVersionAvailable) error {
	url := "https://gcr.io/v2/google_containers/hyperkube-amd64/tags/list"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "build request failed")
	}
	resp, err := v.HttpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	if resp.StatusCode/100 != 2 {
		return errors.New("request status code != 2xx")
	}
	var data struct {
		Tags []string `json:"tags"`
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return errors.Wrap(err, "decode json failed")
	}
	for _, tag := range data.Tags {
		select {
		case <-ctx.Done():
			glog.Infof("context done => return")
			return nil
		case versions <- avro.ApplicationVersionAvailable{
			App:     "Kubernetes",
			Version: tag,
		}:
		}
	}
	return nil
}
