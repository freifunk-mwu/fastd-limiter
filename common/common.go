package common

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	PEERS_CONNECTED = "peers_connected"
	PEER_LIMIT      = "peer_limit"
	KEY             = "key_%s"
)

func ConnectRedis(target string) (conn redis.Conn) {
	conn, err := redis.Dial("tcp", target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
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

func GetPeers(url string) (totalPeers int, err error) {
	metrics, err := getMetrics(url)
	if err != nil {
		return -1, err
	}

	for _, metric := range metrics.Metric {
		totalPeers += int(metric.GetGauge().GetValue())
	}
	return
}
