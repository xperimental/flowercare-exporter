# flowercare-exporter

A [prometheus](https://prometheus.io) exporter which can read data from Xiaomi MiFlora / HHCC Flower Care devices using Bluetooth.

## Installation

First clone the repository, then run the following command to get a binary for your current operating system / architecture. This assumes a working Go installation with modules support (Go >= 1.11.0):

```bash
git clone https://github.com/xperimental/flowercare-exporter.git
cd flowercare-exporter
go build .
```

## Usage

```plain
$ flowercare-exporter -h
Usage of ./flowercare-exporter:
  -i, --adapter string            Bluetooth device to use for communication. (default "hci0")
  -a, --addr string               Address to listen on for connections. (default ":9294")
  -c, --cache-duration duration   Interval during which the results from the Bluetooth device are cached. (default 2m0s)
  -s, --sensor address            MAC-address of sensor to collect data from. Can be specified multiple times.
```

After starting the server will offer the metrics on the `/metrics` endpoint, which can be used as a target for prometheus.

The exporter uses an internal cache, so that each scrape of the exporter does not try to read data from the sensors to avoid unnecessary drain of the battery.

All sensors can optionally have a "name" assigned to them, so they are more easily identifiable in the metrics. This is possible by prefixing the MAC-address with `name=`, for example:

```bash
./flowercare-exporter -s tomatoes=AA:BB:CC:DD:EE:FF
```
