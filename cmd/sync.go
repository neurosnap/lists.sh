package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

func getFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }
        if filepath.Ext(path) != ".md" {
            return nil
        }
		files = append(files, path)
		return nil
	})
	if err != nil {
		return files, err
	}

	return files, nil
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "synconizes a directory with your posts",
	Long: `This command will read a directory you provide and sync it with https://lists.sh.
                Each markdown file inside the directory will be its own post.`,
	RunE: func(cmd *cobra.Command, args []string) error {
        dir, err := filepath.Abs(viper.GetString("source"))
        if err != nil {
			return err
		}
		files, err := getFiles(dir)
		if err != nil {
			return err
		}
        for _, file := range files {
            fmt.Println(file)
        }

        return nil
	},
}
