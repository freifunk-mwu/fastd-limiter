package cmd

import (
	"errors"
	"io/ioutil"
	"regexp"
	"fmt"
	"os"
	"github.com/spf13/viper"
	"github.com/spf13/cobra"
	"github.com/freifunk-mwu/fastd-limiter/common"
)

func findKey(path string, re *regexp.Regexp) (key string, err error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	match := re.FindSubmatch(file)
	if match == nil {
		return "", errors.New("string not found")
	}

	return string(match[1]), err
}

func loadKeys(dirname string) (keys []string, err error) {
	dir, err := os.Open(dirname)
	defer dir.Close()

	if err != nil {
		fmt.Printf( "%v\n", err)
		os.Exit(2)
	}

	files, err := dir.Readdir(-1)

	if err != nil {
		fmt.Printf( "%v\n", err)
		os.Exit(2)
	}

	re := regexp.MustCompile(`key +\"([0-9a-z]{64})\"\;`)

	for _, file := range files {
		if file.IsDir() == false {
			path := fmt.Sprintf("%s/%s", dirname, file.Name())
			key, err := findKey(path, re)
			if err != nil {
				continue
			}
			keys = append(keys, key)
		}
	}
	return
}

// keysCmd represents the keys command
var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "update public keys",
	Run: func(cmd *cobra.Command, args []string) {
		redisDb := viper.GetString("redis_db")
		ttl := viper.GetInt("key_ttl")

		if !viper.IsSet("fastd_keys") {
			fmt.Println("fastd_keys not defined in config file")
			os.Exit(1)
		}
		keysDir := viper.GetString("fastd_keys")

		conn := common.ConnectRedis(redisDb)
		defer conn.Close()

		keys, err := loadKeys(keysDir)

		for _, key := range keys {
			_, err = conn.Do("SET", fmt.Sprintf(common.KEY, key), true, "EX", ttl)
			if err != nil {
				fmt.Printf( "%v\n", err)
				os.Exit(1)
			}
		}

		if verbose {
			fmt.Printf("inserted %d keys with ttl=%d\n", len(keys), ttl)
		}
	},
}

func init() {
	rootCmd.AddCommand(keysCmd)
}
