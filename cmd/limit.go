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
		exporter := viper.GetString("metrics_exporter")

		// check if metrics_url is defined in config
		if !viper.IsSet("metrics_url_local") {
			fmt.Println("metrics_url_local not defined in config file")
			os.Exit(1)
		}
		localMetricsUrl := viper.GetString("metrics_url_local")

		// check if metrics_url is defined in config
		if !viper.IsSet("metrics_url") {
			fmt.Println("metrics_url not defined in config file")
			os.Exit(1)
		}
		metricsUrl := viper.GetString("metrics_url")

		// check if gateways are defined in config
		if !viper.IsSet("gateways") {
			fmt.Println("gateways not defined in config file")
			os.Exit(1)
		}
		gateways := viper.GetStringSlice("gateways")

		// connect to redis server
		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		if exporter == "fastd" {
			limit, peers := common.CalcFastdLimit(localMetricsUrl, metricsUrl, gateways, additional)

			// write peer limit to redis
			_, err := conn.Do("SET", common.PEER_LIMIT, limit)
			if err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(1)
			}

			if verbose {
				fmt.Printf("updated %s to %d\n", common.PEER_LIMIT, limit)
			}

			// write connected peers to redis
			_, err = conn.Do("SET", common.PEERS_CONNECTED, peers)
			if err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(2)
			}

			if verbose {
				fmt.Printf("updated %s to %d\n", common.PEERS_CONNECTED, peers)
			}
		} else if exporter == "kea" {
			// check if subnets are defined in config
			if !viper.IsSet("subnets") {
				fmt.Println("subnets not defined in config file")
				os.Exit(1)
			}
			subnets := viper.GetStringMap("subnets")

			limits, leases := common.CalcDhcpLimits(localMetricsUrl, metricsUrl, gateways)

			for subnet, value := range limits {
				if _, ok := subnets[subnet]; !ok {
					if verbose {
						fmt.Printf("domain for subnet %s not found in config file", subnet)
						continue
					}
				}

				domain := subnets[subnet]

				// write dhcp limit to redis
				limit_key := fmt.Sprintf(common.DHCP_LIMIT, domain)
				_, err := conn.Do("SET", limit_key, value)
				if err != nil {
					fmt.Printf("%v\n", err)
					os.Exit(1)
				}

				if verbose {
					fmt.Printf("updated %s to %d\n", limit_key, value)
				}
			}

			for subnet, value := range leases {
				if _, ok := subnets[subnet]; !ok {
					if verbose {
						fmt.Printf("domain for subnet %s not found in config file", subnet)
						continue
					}
				}

				domain := subnets[subnet]

				// write active leases to redis
				leases_key := fmt.Sprintf(common.DHCP_LEASES, domain)
				_, err := conn.Do("SET", leases_key, value)
				if err != nil {
					fmt.Printf("%v\n", err)
					os.Exit(2)
				}

				if verbose {
					fmt.Printf("updated %s to %d\n", leases_key, value)
				}
			}
		} else {
			fmt.Printf("invalid metrics_exporter: %s\n", exporter)
		}
	},
}

func init() {
	rootCmd.AddCommand(limitCmd)
}
