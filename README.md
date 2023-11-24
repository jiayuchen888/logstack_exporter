# Logstack Exporter

Logstack Exporter is a Prometheus exporter that monitors the presence of a specific log message in Elasticsearch. It periodically checks if the specified log message has occurred within the last few minute.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
  - [Configuration](#configuration)
  - [Building and running](#building-and-running)
- [Using Docker](#using-docker)
- [Metrics](#metrics)

## Features

- Monitors the presence of a specific log message in Elasticsearch.
- Exposes Prometheus metrics for log presence.
- Configurable via a YAML configuration file and environment variables.

## Installation

Logstack Exporter can be run as a standalone binary.

### Configuration 

#### Configuration file

Create a configuration file named `config.yaml` with the following content:

```yaml
scrape_url: "https://your-elasticsearch-host:9200"
scrape_index: "your-index-name"
query_msg: "your-log-message"
```

#### Environment Variables

You have to set the following environment variables:

- ELASTIC_USERNAME: Elasticsearch username.
- ELASTIC_PASSWORD: Elasticsearch password.

Expose the environment variables before running the binary.

### Building and running

```
git clone https://github.com/jiayuchen888/logstack_exporter.git
cd logstack_exporter
make build
./logstack_exporter -config config.yaml
```

By default, Logstack Exporter listens on port 9090 for Prometheus metrics.

## Using Docker

You can build and run the Docker image using the following commands:

```
# Build the Docker image
docker build -t logstack-exporter .

# Run the Docker container
docker run -p 9090:9090 -e ELASTIC_USERNAME=your-username -e ELASTIC_PASSWORD=your-password -v /path/to/your/config-file/folder/:/app/config/ logstack-exporter
```


## Metrics

Logstack Exporter exposes the following Prometheus metric:

- log_presence: Indicates the presence of the specified log message in the last 1 minute. The value is 1 if the log message is present, and 0 otherwise.



