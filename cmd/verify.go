package cmd

import (
	"fmt"
	"os"
	"github.com/spf13/viper"
	"github.com/spf13/cobra"
	"github.com/freifunk-mwu/fastd-limiter/common"
	"github.com/gomodule/redigo/redis"
)

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify <public key>",
	Short: "verify public key",
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		redisDb := viper.GetString("redis_db")
		pubkey := args[0]

		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		exists, _ := redis.Bool(conn.Do("EXISTS", fmt.Sprintf(common.KEY, pubkey)))
		if !exists  {
			if verbose {
				fmt.Println("key not found")
			}
			os.Exit(1)
		}

		peers, err := redis.Int(conn.Do("GET", common.PEERS_CONNECTED))
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		limit, err := redis.Int(conn.Do("GET", common.PEER_LIMIT))
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		if peers >= limit {
			if verbose {
				fmt.Printf("key found but %s execeded: %d > %d\n", common.PEER_LIMIT, peers, limit)
			}
			os.Exit(1)
		}
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
