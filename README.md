# Logstack Exporter

The Logstack Exporter is a Prometheus exporter designed to collect metrics from Elasticsearch logs. It provides insights into log entries, including the timestamp and processing time.

## Features

- Collects metrics from Elasticsearch logs.
- Exposes Prometheus metrics for monitoring and alerting.

Tested on Elasticseach 8.

## Configuration

The Logstack Exporter can be configured using command-line flags. Below are the available configuration options:

- `--scrape_uri`: Elasticsearch endpoint. (Default: `https://127.0.0.1:9092`)
- `--scrape_index`: Elasticsearch index to scrape.
- `--query_msg`: The message to match in the Elasticsearch index.
- `--username`: The username to connect to Elasticsearch for the query. (Can be set using the `ELASTIC_USERNAME` environment variable)
- `--password`: The password of the Elasticsearch user. (Can be set using the `ELASTIC_PASSWORD` environment variable)

Example:
```
./logstack_exporter --scrape_uri=https://your-elasticsearch:9200 --scrape_index=my_logs --query_msg="error" --username=user --password=pass
```

## Using Docker

### Build image

Run the following commands from the project root directory.

```
docker build -t logstack_exporter .
```

### Run
```
docker run -d -p 9090:9090 logstask_exporter \
  --scrape_uri="https://elasticsearch_host:9200" --scrape_index="the-name-of-the-index-you-search" \
  --query_msg="content-of-the-message-you-search" -e ELASTIC_USERNAME=uername -e ELASTIC_PASSWORD=password
```

By default, Logstack Exporter listens on port 9090 for Prometheus metrics.


## Collectors

Exposed metrics:

```
# HELP last_message_received_timestamp timestamp of the last log entry found in Unix seconds
# TYPE last_message_received_timestamp gauge
last_message_received_timestamp 1.700828282e+09
# HELP lostack_processed_time Difference in seconds between log generated and arrival time in Logstash
# TYPE lostack_processed_time gauge
lostack_processed_time 2.165723
```

