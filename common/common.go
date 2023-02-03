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

// define constants
const (
	PEERS_CONNECTED = "fastd_peers_connected"
	PEER_LIMIT      = "fastd_peer_limit"
	DHCP_LIMIT      = "%s_dhcp_limit"
	DHCP_LEASES     = "%s_dhcp_leases"
	KEY             = "key_%s"
)

// helper function to connect to redis
func ConnectRedis(target string) (conn redis.Conn) {
	conn, err := redis.Dial("tcp", target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect failed: %v\n", err)
		os.Exit(1)
	}
	return
}

// get and parse prometheus metrics
func getMetrics(url string, name string) (metrics *dto.MetricFamily, err error) {
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

	if metrics, ok := parsed[name]; ok {
		return metrics, err
	}
	return nil, fmt.Errorf("`%s` not found in metrics", name)
}

// get connected peers from prometheus metrics
func GetFastdPeers(url string) (totalPeers int, err error) {
	metrics, err := getMetrics(url, "fastd_peers_up_total")
	if err != nil {
		return -1, err
	}

	for _, metric := range metrics.Metric {
		totalPeers += int(metric.GetGauge().GetValue())
	}
	return
}

// calculate the fastd peer limit for each gateway
func CalcFastdLimit(localMetricsUrl string, metricsUrl string, gateways []string, additional int) (limit int, localPeers int) {
	// initialize counters
	peersTotal := 0
	gwsOnline := 0
	gwsOffline := 0

	// get locally connected fastd peers
	localPeers, err := GetFastdPeers(localMetricsUrl)
	if err != nil {
		fmt.Printf("%s: %v\n", localMetricsUrl, err)
		gwsOffline++
	} else {
		peersTotal += localPeers
	}

	// get hostname
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(2)
	}

	// get connected peers of configured gateways
	for _, gateway := range gateways {
		if gateway == hostname {
			continue
		}

		url := fmt.Sprintf(metricsUrl, gateway)
		peers, err := GetFastdPeers(url)
		if err != nil {
			fmt.Printf("%s: %v\n", url, err)
			gwsOffline++
		} else {
			peersTotal += peers
		}
	}

	gwsOnline = len(gateways) - gwsOffline

	// calculate limit
	peersAvg := peersTotal / gwsOnline
	gauge := (1 + gwsOffline) * additional
	limit = peersAvg + gauge

	return
}

// get dhcp leases from prometheus metrics
func GetDhcpLeases(url string) (leases map[string]int, err error) {
	leases = make(map[string]int)

	assignedLeasesMetrics, err := getMetrics(url, "kea_dhcp4_addresses_assigned_total")
	if err != nil {
		return nil, err
	}

	for _, metric := range assignedLeasesMetrics.Metric {
		labels := metric.GetLabel()

		var subnet string

		for _, label := range labels {
			if label.GetName() == "subnet" {
				subnet = label.GetValue()
				break
			}
		}
		leases[subnet] = int(metric.GetGauge().GetValue())
	}

	return
}

// calculate the fastd peer limit for each gateway
func CalcDhcpLimits(localMetricsUrl string, metricsUrl string, gateways []string) (limits map[string]int, localLeases map[string]int) {
	limits = make(map[string]int)
	localLeases = make(map[string]int)
	leasesTotal := make(map[string]int)
	gwsOnline := 0
	gwsOffline := 0

	// get local dhcp leases
	localLeases, err := GetDhcpLeases(localMetricsUrl)
	if err != nil {
		fmt.Printf("%s: %v\n", localMetricsUrl, err)
		gwsOffline++
	} else {
		for subnet, value := range localLeases {
			leasesTotal[subnet] += value
		}
	}

	// get hostname
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(2)
	}

	// get dhcp leases of configured gateways
	for _, gateway := range gateways {
		// skip if gateway name matches hostname
		if gateway == hostname {
			continue
		}

		url := fmt.Sprintf(metricsUrl, gateway)
		leases, err := GetDhcpLeases(url)
		if err != nil {
			fmt.Printf("%s: %v\n", url, err)
			gwsOffline++
		} else {
			for subnet, value := range leases {
				leasesTotal[subnet] += value
			}
		}
	}

	gwsOnline = len(gateways) - gwsOffline

	// calculate limit
	for subnet, value := range leasesTotal {
		limits[subnet] = value / gwsOnline
	}

	return
}
