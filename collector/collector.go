// Copyright (c) 2015 neezgee
//
// Licensed under the MIT license: https://opensource.org/licenses/MIT
// Permission is granted to use, copy, modify, and redistribute the work.
// Full license information available in the project LICENSE file.
//

package collector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/olivere/elastic/v7"
	"github.com/prometheus/client_golang/prometheus"
)

type Exporter struct {
	URI         string
	scrapeIndex string
	queryMsg    string
	username    string
	password    string
	mutex       sync.Mutex

	scrapeFailures prometheus.Counter
	lastTimestamp  prometheus.Gauge
	processedTime  prometheus.Gauge
	logger         log.Logger
}

type Config struct {
	ScrapeURI   string
	ScrapeIndex string
	QueryMsg    string
	Username    string
	Password    string
}

func NewExporter(logger log.Logger, config *Config) *Exporter {
	return &Exporter{
		URI:         config.ScrapeURI,
		scrapeIndex: config.ScrapeIndex,
		queryMsg:    config.QueryMsg,
		username:    config.Username,
		password:    config.Password,
		logger:      logger,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "exporter_scrape_failures_total",
			Help: "Number of errors while scraping elasticsearch index",
		}),
		lastTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "last_message_received_timestamp",
			Help: "timestamp of the last log entry found in Unix seconds",
		},
		),
		processedTime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "lostack_processed_time",
			Help: "Difference in seconds between log generated and arrival time in Logstash",
		},
		),
	}
}

// Describe implements Prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.scrapeFailures.Describe(ch)
	e.lastTimestamp.Describe(ch)
	e.processedTime.Describe(ch)
}

func newElasticsearchClient(uri, username, password string) (*elastic.Client, error) {
	// Create a custom http.Client to skip SSL verification
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Create the Elasticsearch client with the custom http.Client
	client, err := elastic.NewClient(
		elastic.SetURL(uri),
		elastic.SetBasicAuth(username, password),
		elastic.SetHttpClient(httpClient),
		elastic.SetSniff(false),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type HitsInfo struct {
	Total int `json:"total"`
	Hits  struct {
		Hits []struct {
			Source struct {
				Timestamp           string `json:"@timestamp"`
				LogstashProcessedAt string `json:"logstash_processed_at"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {
	// Configure Elasticsearch client
	client, err := newElasticsearchClient(e.URI, e.username, e.password)
	if err != nil {
		return fmt.Errorf("error building connection to Elasticseach : %v", err)
	}
	// Create a match query for the specific log message
	matchQuery := elastic.NewMatchQuery("message", e.queryMsg)

	// Execute the search query
	ctx := context.Background()
	searchResult, err := client.Search(e.scrapeIndex).
		Query(matchQuery).
		Size(1).
		Sort("@timestamp", false).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("error scraping Elastcisearch index: %v", err)
	}

	// Marshal the SearchResult into a JSON byte slice
	searchResultBytes, err := json.Marshal(searchResult)
	if err != nil {
		return fmt.Errorf("error marshaling search result to JSON: %v", err)
	}

	// Unmarshal the JSON into the HitsInfo struct
	var result HitsInfo
	err = json.Unmarshal(searchResultBytes, &result)
	if err != nil {
		return fmt.Errorf("error unmarshaling JSON to HitsInfo: %v", err)
	}

	// Access the fields as needed
	if len(result.Hits.Hits) > 0 {
		// Now timestamp and logstashProcessedAt contain the extracted values
		timestamp := result.Hits.Hits[0].Source.Timestamp
		logstashProcessedAt := result.Hits.Hits[0].Source.LogstashProcessedAt

		// Parse the timestamp into a time.Time object
		parsedTimestamp, err := time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			return fmt.Errorf("error parsing timestamp: %v", err)
		}

		// Set the lastTimestamp gauge to the Unix timestamp of the parsed time
		e.lastTimestamp.Set(float64(parsedTimestamp.Unix()))

		// Parse the logstashProcessedAt into a time.Time object
		parsedLogstashProcessedAt, err := time.Parse(time.RFC3339Nano, logstashProcessedAt)
		if err != nil {
			return fmt.Errorf("error parsing logstash_processed_at: %v", err)
		}

		// Calculate the difference between parsedTimestamp and parsedLogstashProcessedAt
		processedTime := parsedLogstashProcessedAt.Sub(parsedTimestamp).Seconds()

		// Set the processedTime gauge to the calculated value
		e.processedTime.Set(processedTime)

		// Register the metrics with Prometheus
		e.lastTimestamp.Collect(ch)
		e.processedTime.Collect(ch)
	}

	return nil
}

// Collect implements Prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		level.Error(e.logger).Log("msg", "Error scraping Elasticsearch Index:", "err", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	return
}
