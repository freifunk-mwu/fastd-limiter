package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fastd-limiter",
	Short: "limit fastd peer connections per gateway",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// flags and configuration settings
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/fastd-limiter.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "be verbose")
}

// initConfig reads in config file
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config file in /etc
		viper.AddConfigPath("/etc/")
		viper.SetConfigName("fastd-limiter")
	}

	viper.SetDefault("additional", 8)
	viper.SetDefault("redis_db", ":6379")
	viper.SetDefault("metrics_url_local", "http://127.0.0.1:9281/metrics")
	viper.SetDefault("key_ttl", 900)

	// If a config file is found, read it in.
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}
