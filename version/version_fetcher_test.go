// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bborbe/kafka-k8s-version-collector/avro"
	"github.com/bborbe/kafka-k8s-version-collector/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Version Fetcher", func() {
	var fetcher version.Fetcher
	var server *ghttp.Server
	BeforeEach(func() {
		server = ghttp.NewServer()
		fetcher = version.NewFetcher(
			http.DefaultClient,
			server.URL(),
		)
	})
	AfterEach(func() {
		server.Close()
	})

	It("returns nothing if empty list", func() {
		server.RouteToHandler(http.MethodGet, "/v2/google_containers/hyperkube-amd64/tags/list", func(resp http.ResponseWriter, req *http.Request) {
			fmt.Fprint(resp, `{}`)
		})
		versions := make(chan avro.ApplicationVersionAvailable)
		var list []avro.ApplicationVersionAvailable
		go func() {
			defer close(versions)
			err := fetcher.Fetch(context.Background(), versions)
			Expect(err).NotTo(HaveOccurred())
		}()
		for version := range versions {
			list = append(list, version)
		}
		Expect(list).To(HaveLen(0))
	})
	It("returns versions", func() {
		server.RouteToHandler(http.MethodGet, "/v2/google_containers/hyperkube-amd64/tags/list", func(resp http.ResponseWriter, req *http.Request) {
			fmt.Fprint(resp, `{"tags":["v1","v2","v3"]}`)
		})
		versions := make(chan avro.ApplicationVersionAvailable)
		var list []avro.ApplicationVersionAvailable
		go func() {
			defer close(versions)
			err := fetcher.Fetch(context.Background(), versions)
			Expect(err).NotTo(HaveOccurred())
		}()
		for version := range versions {
			list = append(list, version)
		}
		Expect(list).To(HaveLen(3))
		Expect(list[0].App).To(Equal("Kubernetes"))
		Expect(list[0].Version).To(Equal("v1"))
		Expect(list[1].App).To(Equal("Kubernetes"))
		Expect(list[1].Version).To(Equal("v2"))
		Expect(list[2].App).To(Equal("Kubernetes"))
		Expect(list[2].Version).To(Equal("v3"))
	})
	It("returns an error if not valid json", func() {
		server.RouteToHandler(http.MethodGet, "/v2/google_containers/hyperkube-amd64/tags/list", func(resp http.ResponseWriter, req *http.Request) {
			fmt.Fprint(resp, `asdf`)
		})
		versions := make(chan avro.ApplicationVersionAvailable)
		defer close(versions)
		err := fetcher.Fetch(context.Background(), versions)
		Expect(err).To(HaveOccurred())
	})
	It("returns an error if status not 2xx", func() {
		server.RouteToHandler(http.MethodGet, "/v2/google_containers/hyperkube-amd64/tags/list", func(resp http.ResponseWriter, req *http.Request) {
			resp.WriteHeader(http.StatusNotFound)
		})
		versions := make(chan avro.ApplicationVersionAvailable)
		defer close(versions)
		err := fetcher.Fetch(context.Background(), versions)
		Expect(err).To(HaveOccurred())
	})
	It("returns an error if http do fails", func() {
		fetcher = version.NewFetcher(
			&http.Client{
				Transport: &ErrorRoundTripper{},
			},
			server.URL(),
		)
		versions := make(chan avro.ApplicationVersionAvailable)
		defer close(versions)
		err := fetcher.Fetch(context.Background(), versions)
		Expect(err).To(HaveOccurred())
	})
	It("returns without error if context is cancel", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		server.RouteToHandler(http.MethodGet, "/v2/google_containers/hyperkube-amd64/tags/list", func(resp http.ResponseWriter, req *http.Request) {
			cancel()
			fmt.Fprint(resp, `{"tags":["v1","v2","v3"]}`)
		})
		versions := make(chan avro.ApplicationVersionAvailable)
		err := fetcher.Fetch(ctx, versions)
		Expect(err).To(BeNil())
	})
})
