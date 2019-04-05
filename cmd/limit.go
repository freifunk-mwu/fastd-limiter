package cmd

import (
	"fmt"
	"github.com/freifunk-mwu/fastd-limiter/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

// limitCmd represents the limit command
var limitCmd = &cobra.Command{
	Use:   "limit",
	Short: "update peer limit",
	Run: func(cmd *cobra.Command, args []string) {
		// get config vars
		additional := viper.GetInt("additional")
		redisDb := viper.GetString("redis_db")
		localMetricsUrl := viper.GetString("metrics_url_local")

		// check if metrics_url is defined in config
		if !viper.IsSet("metrics_url") {
			fmt.Println("metrics_url not defined in config file")
			os.Exit(1)
		}
		metricsUrl := viper.GetString("metrics_url")

		// check if gateways is defined in config
		if !viper.IsSet("gateways") {
			fmt.Println("gateways not defined in config file")
			os.Exit(1)
		}
		gateways := viper.GetStringSlice("gateways")

		// connect to redis server
		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		// initialize counters
		peersTotal := 0
		gwsOnline := 0
		gwsOffline := 0

		// get locally connected peers
		peers, err := common.GetPeers(localMetricsUrl)
		if err != nil {
			fmt.Printf("%s: %v\n", localMetricsUrl, err)
			gwsOffline++
		} else {
			gwsOnline++
			peersTotal += peers
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
			peers, err = common.GetPeers(url)
			if err != nil {
				fmt.Printf("%s: %v\n", url, err)
				gwsOffline++
			} else {
				gwsOnline++
				peersTotal += peers
			}
		}

		// calculate limit
		peersAvg := peersTotal / gwsOnline
		gauge := (1 + gwsOffline) * additional
		limit := peersAvg + gauge

		// write peer limit to redis
		_, err = conn.Do("SET", common.PEER_LIMIT, limit)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		if verbose {
			fmt.Printf("updated %s to %d\n", common.PEER_LIMIT, limit)
		}
	},
}

func init() {
	rootCmd.AddCommand(limitCmd)
}
