package main

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/naoina/toml"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const (
	PEERS_CONNECTED = "peers_connected"
	PEER_LIMIT      = "peer_limit"
	KEY             = "key_%s"
)

type Config struct {
	Additional int      `toml:"additional"`
	Metrics    string   `toml:"metrics_url"`
	Redis      string   `toml:"redis_db"`
	Keys       string   `toml:"fastd_keys"`
	Timeout    int      `toml:"key_ttl"`
	Gateways   []string `toml:"gateways"`
}

func loadConfig(configPath string) (config *Config) {
	config, err := readConfigFile(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to load config file:", err)
		os.Exit(2)
	}
	return
}

func readConfigFile(path string) (config *Config, err error) {
	config = &Config{}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(file, config)
	if err != nil {
		return nil, err
	}
	return
}

func findString(path string, re *regexp.Regexp) (key string, err error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	match := re.FindSubmatch(file)
	if match == nil {
		return "", errors.New("string not found")
	}

	return string(match[1]), err
}

func findKeys(dirname string) (keys []string, err error) {
	dir, err := os.Open(dirname)
	defer dir.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	files, err := dir.Readdir(-1)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	re := regexp.MustCompile(`key +\"([0-9a-z]{64})\"\;`)

	for _, file := range files {
		if file.IsDir() == false {
			path := fmt.Sprintf("%s/%s", dirname, file.Name())
			key, err := findString(path, re)
			if err != nil {
				continue
			}
			keys = append(keys, key)
		}
	}
	return
}

func connectRedis(target string) (conn redis.Conn) {
	conn, err := redis.Dial("tcp", target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed:\n%v\n", err)
		os.Exit(2)
	}
	return
}

func getMetrics(url string) (metrics *dto.MetricFamily, err error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	} else if res.StatusCode != 200 {
		return metrics, errors.New(res.Status)
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	body := string(bodyBytes)

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	if metrics, ok := parsed["fastd_peers_up_total"]; ok {
		return metrics, err
	}
	return nil, errors.New("`fastd_peers_up_total` not found in metrics")
}

func getPeers(url string) (totalPeers int, err error) {
	metrics, err := getMetrics(url)
	if err != nil {
		return -1, err
	}

	for _, metric := range metrics.Metric {
		totalPeers += int(metric.GetGauge().GetValue())
	}
	return
}

func updateKeys(config *Config) {
	conn := connectRedis(config.Redis)
	defer conn.Close()

	keys, err := findKeys(config.Keys)

	for _, key := range keys {
		_, err = conn.Do("SET", fmt.Sprintf(KEY, key), true, "EX", config.Timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
}

func updatePeers(config *Config) {
	conn := connectRedis(config.Redis)
	defer conn.Close()

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	url := fmt.Sprintf(config.Metrics, hostname)

	peers, err := getPeers(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", url, err)
		os.Exit(2)
	}

	_, err = conn.Do("SET", PEERS_CONNECTED, peers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	return
}

func limitPeers(config *Config) {
	conn := connectRedis(config.Redis)
	defer conn.Close()

	peersTotal := 0
	gwsOnline := 0
	gwsOffline := 0

	for _, gateway := range config.Gateways {
		url := fmt.Sprintf(config.Metrics, gateway)
		peers, err := getPeers(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", url, err)
			gwsOffline++
		} else {
			gwsOnline++
			peersTotal += peers
		}
	}

	peersAvg := peersTotal / gwsOnline
	gauge := (1 + gwsOffline) * config.Additional
	limit := peersAvg + gauge

	_, err := conn.Do("SET", PEER_LIMIT, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	return
}

func verifyPeer(config *Config, pubkey string) int {
	conn := connectRedis(config.Redis)
	defer conn.Close()

	_, err := redis.Bool(conn.Do("EXISTS", fmt.Sprintf(KEY, pubkey)))
	if err != nil {
		return 2
	}

	peers, err := redis.Int(conn.Do("GET", PEERS_CONNECTED))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	limit, err := redis.Int(conn.Do("GET", PEER_LIMIT))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	if peers >= limit {
		return 2
	}

	return (0)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: fastd-peer-limit <command> [<args>]")
		fmt.Println("  keys: update public keys")
		fmt.Println("  limit: update peer limit")
		fmt.Println("  peers: update connected peers")
		fmt.Println("  verify: verify peer")
		os.Exit(2)
	}

	command := os.Args[1]
	config := loadConfig("/etc/fastd-limiter.cfg")

	switch command {
	case "keys":
		updateKeys(config)
	case "limit":
		limitPeers(config)
	case "peers":
		updatePeers(config)
	case "verify":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Please supply the public key to verify.")
			os.Exit(2)
		}

		key := os.Args[2]

		if keyLen := len(key); keyLen != 64 {
			fmt.Fprintf(os.Stderr, "invalid key length: %d != 64\n", keyLen)
			os.Exit(2)
		}

		ret := verifyPeer(config, key)
		os.Exit(ret)
	default:
		fmt.Fprintf(os.Stderr, "%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
}
