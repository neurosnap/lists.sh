package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringP("source", "s", "", "source directory to sync with")

	err := viper.BindPFlags(syncCmd.Flags())
	if err != nil {
		log.Println("Unable to bind pflags:", err)
	}
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "synconizes a directory with your posts",
	Long: `This command will read a directory you provide and sync it with https://lists.sh.
                Each markdown file inside the directory will be its own post.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("SYNC")
		fmt.Printf("%s\n", viper.GetString("source"))
	},
}
