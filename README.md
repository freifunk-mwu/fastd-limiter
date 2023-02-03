# fastd-limiter

We are building a peer limiter for the
[fastd](https://projects.universe-factory.net/projects/fastd/wiki) vpn daemon.

## Getting Started

### Requirements

* [Redis](https://redis.io/)
* [fastd-exporter](https://github.com/freifunk-darmstadt/fastd-exporter) or
* [kea-exporter](https://github.com/mweinelt/kea-exporter/tree/develop/kea_exporter)

### Installation

```
$ go get github.com/freifunk-mwu/fastd-limiter
```

### Configuration

fastd-limiter searches for a config file in `/etc/fastd-limiter.yaml`. An example
can be found in this repository.

These are the mandatory config options: `fastd_keys`, `gateways`, `metrics_url_local`,
`metrics_url`. If `metrics_exporter` is set to _kea_ option `subnets` is also mandatory.

**fastd_keys** is the path to the directory that holds the fastd public keys.

**gateways** is a list of all gateways from which *fastd-exporter* metrics should
be retrieved.

**subnets** is a table of all IPv4 subnets and its domain codes for which
*fastd-exporter* metrics should be retrieved.

**local_metrics_url** is the url for retrieving local metrics.

**metrics_url** is the base url for retrieving metrics. It has to include on one
placeholder *(%s)* which is replaced with the values from **gateways**.

*All other options are populated with sane default values. See example config
for more information.*

### Usage

fastd-limiter has three commands: `keys`, `limit`, and `verify`. The
first two are meant be called periodically. `verify` is supposed to be called
via `on verify` by fastd.

```
./fastd-limiter <command>
```

**keys** reads all keys found into the Redis database with the configured TTL.

**limit** reads the Prometheus metrics from all configured gateways, calculates
the average peer count and stores it in the Redis database.

**verify** takes the public key to verify as first command line argument and
checks if it is present in the Redis database and if the connected locally
connected peers are below the calculated limit. If one of the two criteria is
not met the application exits with a return code of 1. If `metrics_exporter` is
set to _kea_ the domain code is required as second argument.

```
./fastd-limiter verify <fastd_key> [<domain>]
```

### Help

Check out the help for additional information.

```
$ ./fastd-limiter --help
```

### Formatting

We are using go's internal formatting for this codebase.

```
$ go fmt
```
