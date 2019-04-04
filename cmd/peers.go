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
		metricsUrl := viper.GetString("metrics_url_local")

		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		peers, err := common.GetPeers(metricsUrl)
		if err != nil {
			fmt.Printf("%s: %v\n", metricsUrl, err)
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
