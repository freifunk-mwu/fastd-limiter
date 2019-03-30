package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/viper"
	"github.com/spf13/cobra"
	"github.com/freifunk-mwu/fastd-limiter/common"
)

// peersCmd represents the peers command
var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "update connected peers",
	Run: func(cmd *cobra.Command, args []string) {
		redisDb := viper.GetString("redis_db")

		if !viper.IsSet("metrics_url") {
			fmt.Println("metrics_url not defined in config file")
			os.Exit(1)
		}
		metricsUrl := viper.GetString("metrics_url")

		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		hostname, err := os.Hostname()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(2)
		}
		url := fmt.Sprintf(metricsUrl, hostname)

		peers, err := common.GetPeers(url)
		if err != nil {
			fmt.Printf("%s: %v\n", url, err)
			os.Exit(2)
		}

		_, err = conn.Do("SET", common.PEERS_CONNECTED, peers)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(2)
		}

		if verbose {
			fmt.Printf("updated %s to %d\n", common.PEERS_CONNECTED, peers)
		}
	},
}

func init() {
	rootCmd.AddCommand(peersCmd)
}
