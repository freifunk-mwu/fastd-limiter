package cmd

import (
	"os"
	"fmt"
	"github.com/freifunk-mwu/fastd-limiter/common"
	"github.com/spf13/viper"
	"github.com/spf13/cobra"
)

// limitCmd represents the limit command
var limitCmd = &cobra.Command{
	Use:   "limit",
	Short: "update peer limit",
	Run: func(cmd *cobra.Command, args []string) {
		additional := viper.GetInt("additional")
		redisDb := viper.GetString("redis_db")
		localMetricsUrl := viper.GetString("metrics_url_local")

		if !viper.IsSet("metrics_url") {
			fmt.Println("metrics_url not defined in config file")
			os.Exit(1)
		}
		metricsUrl := viper.GetString("metrics_url")

		if !viper.IsSet("gateways") {
			fmt.Println("gateways not defined in config file")
			os.Exit(1)
		}
		gateways := viper.GetStringSlice("gateways")

		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		peersTotal := 0
		gwsOnline := 0
		gwsOffline := 0

		peers, err := common.GetPeers(localMetricsUrl)
		if err != nil {
			fmt.Printf("%s: %v\n", localMetricsUrl, err)
			gwsOffline++
		} else {
			gwsOnline++
			peersTotal += peers
		}

		hostname, err := os.Hostname()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(2)
		}

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

		peersAvg := peersTotal / gwsOnline
		gauge := (1 + gwsOffline) * additional
		limit := peersAvg + gauge

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
