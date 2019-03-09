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

//go:generate counterfeiter -o ../mocks/fetcher.go --fake-name Fetcher . Fetcher
type Fetcher interface {
	Fetch(ctx context.Context, versions chan<- avro.ApplicationVersionAvailable) error
}

func NewFetcher(
	httpClient *http.Client,
	url string,
) Fetcher {
	return &fetcher{
		httpClient: httpClient,
		url:        url,
	}
}

type fetcher struct {
	httpClient *http.Client
	url        string
}

func (f *fetcher) Fetch(ctx context.Context, versions chan<- avro.ApplicationVersionAvailable) error {
	url := f.url + "/v2/google_containers/hyperkube-amd64/tags/list"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "build request failed")
	}
	glog.V(1).Infof("%s %s", req.Method, req.URL.String())
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return errors.New("request status code != 2xx")
	}
	var data struct {
		Tags []string `json:"tags"`
	}
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
