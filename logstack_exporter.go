// Copyright (c) 2015 neezgee
//
// Licensed under the MIT license: https://opensource.org/licenses/MIT
// Permission is granted to use, copy, modify, and redistribute the work.
// Full license information available in the project LICENSE file.
//

package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"logstack_exporter/collector"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var (
	scrapeURI    = kingpin.Flag("scrape_uri", "elasticsearch endpoint").Default("https://127.0.0.1:9092").String()
	scrapeIndex  = kingpin.Flag("scrape_index", "elasticsearch index").Default("").String()
	queryMsg     = kingpin.Flag("query_msg", "the message to match in elasticsearch index").Default("").String()
	username     = kingpin.Flag("username", "The username connect to elasticsearch for the query").Default("").Envar("ELASTIC_USERNAME").String()
	password     = kingpin.Flag("password", "The password of elasticsearch user").Default("").Envar("ELASTIC_PASSWORD").String()
	toolkitFlags = kingpinflag.AddFlags(kingpin.CommandLine, ":9090")
	gracefulStop = make(chan os.Signal, 1)
)

func main() {
	promlogConfig := &promlog.Config{}

	// Parse flags
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Print("logstack_exporter"))
	kingpin.Parse()
	logger := promlog.New(promlogConfig)
	// listen to termination signals from the OS
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	signal.Notify(gracefulStop, syscall.SIGHUP)
	signal.Notify(gracefulStop, syscall.SIGQUIT)

	config := &collector.Config{
		ScrapeURI:   *scrapeURI,
		ScrapeIndex: *scrapeIndex,
		QueryMsg:    *queryMsg,
		Username:    *username,
		Password:    *password,
	}

	exporter := collector.NewExporter(logger, config)
	prometheus.MustRegister(exporter)

	level.Info(logger).Log("msg", "Starting logstack_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build", version.BuildContext())
	level.Info(logger).Log("msg", "Collect from: ", "scrape_uri", *scrapeURI)

	// listener for the termination signals from the OS
	go func() {
		level.Info(logger).Log("msg", "listening and wait for graceful stop")
		sig := <-gracefulStop
		level.Info(logger).Log("msg", "caught sig: %+v. Wait 2 seconds...", "sig", sig)
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{}
	if err := web.ListenAndServe(server, toolkitFlags, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
