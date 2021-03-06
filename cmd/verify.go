package cmd

import (
	"fmt"
	"github.com/freifunk-mwu/fastd-limiter/common"
	"github.com/gomodule/redigo/redis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify <public key>",
	Short: "verify public key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// get config var
		redisDb := viper.GetString("redis_db")

		// get public key to verify from arguments
		pubkey := args[0]

		// connect to redis server
		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		// check if key exists in redis
		// reject (exit) if key is not present
		exists, _ := redis.Bool(conn.Do("EXISTS", fmt.Sprintf(common.KEY, pubkey)))
		if !exists {
			if verbose {
				fmt.Println("key not found")
			}
			os.Exit(1)
		}

		// get locally connected peers from redis
		peers, err := redis.Int(conn.Do("GET", common.PEERS_CONNECTED))
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		// get peer limit from redis
		limit, err := redis.Int(conn.Do("GET", common.PEER_LIMIT))
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		// reject (exit) if connected peers are higher or equal limit
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
