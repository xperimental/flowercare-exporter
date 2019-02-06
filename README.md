# flowercare-exporter

A [prometheus](https://prometheus.io) exporter which can read data from Xiaomi MiFlora / HHCC Flower Care devices using Bluetooth.

It uses the `gatttool` from BlueZ to do the actual Bluetooth communication, so it probably only works on Linux. Getting rid of this dependency and doing the communication directly in Go would be great, especially for portability.

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
Usage of flowercare-exporter:
  -i, --adapter string   Bluetooth device to use for communication. (default "hci0")
  -a, --addr string      Address to listen on for connections. (default ":9294")
  -b, --device string    MAC-Address of Flower Care device.
```

After starting the server will offer the metrics on the `/metrics` endpoint, which can be used as a target for prometheus.

The exporter will query the bluetooth device every time it is scraped by prometheus. The default scrape interval is usually not necessary, because the data does not change this frequently. I recommend setting a longer `scrape_interval`, so that the battery of your device lasts longer:

```yml
scrape_configs:
- job_name: 'flowercare'
  scrape_interval: 120s
  static_configs:
  - targets: ['localhost:9294']
```
