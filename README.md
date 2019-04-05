# fastd-limiter

We are building a peer limiter for the
[fastd](https://projects.universe-factory.net/projects/fastd/wiki) vpn daemon.

## Getting Started

### Requirements

* [Redis](https://redis.io/)
* [fastd-exporter](https://github.com/freifunk-darmstadt/fastd-exporter)

### Installation

```
$ go get github.com/freifunk-mwu/fastd-limiter
```

### Configuration

fastd-limiter searches for a config file in `/etc/fastd-limiter.yaml`. An example
can be found in this repository.

There are three mandatory config options: `fastd_keys`, `gateways` and
`metrics_url`.

**fastd_keys** is the path to the directory that holds the fastd public keys.

**gateways** is a list of all gateways from which *fastd-exporter* metrics should
be retrieved.

**metrics_url** is the base url for retrieving metrics. It has to include on one
placeholder *(%s)* which is replaced with the values from **gateways**.

*All other options are populated with sane default values. See example config
for more information.*

### Usage

fastd-limiter has four commands: `keys`, `limit`, `peers` and `verify`. The
first three are meant be called periodically. `verify` is supposed to be called
via `on verify` by fastd.

```
./fastd-limiter [command]
```

**keys** reads all keys found into the Redis database with the configured TTL.

**limit** reads the prometheus metrics from all configured gateways, calculates
the average peer count and stores it in the Redis database.

**peers** reads the local prometheus metrics and writes the locally connected
peers count into the Redis database.

**verify** takes the public key to verify as first command line argument and
checks if it is present in the Redis database and if the connected locally
connected peers are below the calculated limit. If one of the two criteria is
not met the application exits with a return code of 1.

```
./fastd-limiter verify [fastd_key]
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
