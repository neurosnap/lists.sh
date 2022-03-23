package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "lists.sh",
		Short: "microblog for your lists",
		Long: `A fast and easy way to publish your lists
                Complete documentation is available at https://lists.sh`,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
			fmt.Println("root command")
		},
	}
	client = &http.Client{}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/lists.sh/config.yaml)")
	rootCmd.PersistentFlags().StringP("url", "u", "", "api url for lists.sh")
	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		log.Println("Unable to bind pflags:", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$XDG_CONFIG_HOME/lists.sh")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
