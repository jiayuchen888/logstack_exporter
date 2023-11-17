package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"time"
  "flag"

	"github.com/fsnotify/fsnotify"
  "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/olivere/elastic/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/spf13/viper"
)

var (
  logger = log.NewLogfmtLogger(os.Stderr)

	logPresence = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "log_presence",
			Help: "Indicates the presence of a specific log message in the last 5 minutes",
		},
	)
)

func init() {
	prometheus.MustRegister(logPresence)
}

type Config struct {
	ElasticsearchURL string `mapstructure:"elasticsearch_url"`
	IndexName        string `mapstructure:"index_name"`
	LogMessage       string `mapstructure:"log_message"`
	ElasticUsername  string `mapstructure:"elastic_username"`
	ElasticPassword  string `mapstructure:"elastic_password"`
}

func main() {
	// Parse command-line arguments
  configFilePath := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	// Load configuration
	config := loadConfig(*configFilePath)

	// Configure Elasticsearch client
	client, err := newElasticsearchClient(config)
	if err != nil {
    level.Error(logger).Log("msg","Error creating Elasticsearch client:","err", err)
		os.Exit(1)
	}

  level.Info(logger).Log("msg","Starting logstack_exporter","version",version.Info())
  level.Info(logger).Log("msg","Scarping elasticsearch","endpoint", config.ElasticsearchURL)

	// HTTP handler for Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())

	// Goroutine to periodically check log presence
	go func() {
		for {
			err := checkLogPresence(client, config)
			if err != nil {
        level.Error(logger).Log("msg", "Error", "err", err)
			}

			time.Sleep(5 * time.Minute)
		}
	}()

  http.ListenAndServe(":9090", nil)

}

func loadConfig(filePath string) Config {
	// Set up configuration using viper
	viper.SetConfigFile(filePath)

	// Define default values
	viper.SetDefault("elasticsearch_url", "http://your-elasticsearch-host:9200")
	viper.SetDefault("index_name", "your-index-name")
	viper.SetDefault("log_message", "your-log-message")

  // Check for the presence of environment variables for the username and password
	if username := os.Getenv("ELASTIC_USERNAME"); username != "" {
		viper.Set("elastic_username", username)
	} else {
		viper.SetDefault("elastic_username", "your-elastic-username")
	}

  	// Check for the presence of an environment variable for the password
	if password := os.Getenv("ELASTIC_PASSWORD"); password != "" {
		viper.Set("elastic_password", password)
	} else {
		viper.SetDefault("elastic_password", "your-elastic-password")
	}

	// Read configuration file
	err := viper.ReadInConfig()
	if err != nil {
    level.Error(logger).Log("msg", "Error reading config file", "err", err)
		os.Exit(1)
	}

	// Unmarshal configuration into struct
	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
    level.Error(logger).Log("msg", "Error unmarshaling config", "err", err)
		os.Exit(1)
	}

	// Watch for changes in the config file
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
    level.Info(logger).Log("msg", "Config file changed", "file", e.Name)
		// Reload the configuration if it changes
		viper.Unmarshal(&config)
	})

	return config
}

func newElasticsearchClient(config Config) (*elastic.Client, error) {
	// Create a custom http.Client to skip SSL verification
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Create the Elasticsearch client with the custom http.Client
	client, err := elastic.NewClient(
		elastic.SetURL(config.ElasticsearchURL),
		elastic.SetBasicAuth(config.ElasticUsername, config.ElasticPassword),
		elastic.SetHttpClient(httpClient),
    elastic.SetSniff(false),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func checkLogPresence(client *elastic.Client, config Config) error {
	// Create a range query for the last 5 minutes
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	rangeQuery := elastic.NewRangeQuery("@timestamp").
		Gte(fiveMinutesAgo.Format(time.RFC3339)).
		Lte(now.Format(time.RFC3339))

	// Create a match query for the specific log message
	matchQuery := elastic.NewMatchQuery("message", config.LogMessage)

	// Combine the range and match queries
	boolQuery := elastic.NewBoolQuery().Must(rangeQuery, matchQuery)

	// Execute the search query
	ctx := context.Background()
	searchResult, err := client.Search(config.IndexName).
		Query(boolQuery).
		Size(1).
		Sort("@timestamp", false).
		Do(ctx)
	if err != nil {
		return err
	}

  logPresence.Set(float64(searchResult.TotalHits()))

	return nil
}
