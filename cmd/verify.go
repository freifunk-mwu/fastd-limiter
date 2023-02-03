package cmd

import (
	"fmt"
	"github.com/freifunk-mwu/fastd-limiter/common"
	"github.com/gomodule/redigo/redis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"time"
)

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify <public key> [<domain>]",
	Short: "verify public key",
	Args:  cobra.MatchAll(cobra.MinimumNArgs(1), cobra.MaximumNArgs(2)),
	Run: func(cmd *cobra.Command, args []string) {
		// get config var
		redisDb := viper.GetString("redis_db")
		exporter := viper.GetString("metrics_exporter")

		// get public key to verify from arguments
		pubkey := args[0]

		// require second argument when running using kea metrics
		var domain string
		if exporter == "kea" {
			if len(args) > 1 {
				domain = args[1]
			} else {
				fmt.Println("error: missing argument `domain`")
				os.Exit(1)
			}
		}

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

		if exporter == "fastd" {
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
		} else if exporter == "kea" {
			// get local dhcp leases from redis
			leases_key := fmt.Sprintf(common.DHCP_LEASES, domain)
			leases, err := redis.Int(conn.Do("GET", leases_key))
			if err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(1)
			}

			// get dhcp limit from redis
			limit_key := fmt.Sprintf(common.DHCP_LIMIT, domain)
			limit, err := redis.Int(conn.Do("GET", limit_key))
			if err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(1)
			}

			// respond with delay if local leases are higher or equal limit
			if leases >= limit {
				if verbose {
					fmt.Printf("key found but %s execeded: %d > %d, delaying response\n", limit_key, leases, limit)
				}
				time.Sleep(5 * time.Second)
			}
			os.Exit(0)
		} else {
			fmt.Printf("invalid metrics_exporter: %s\n", exporter)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
