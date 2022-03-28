package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
    "bytes"
    "encoding/json"
    "net/http"

	"github.com/neurosnap/lists.sh/api"
	"github.com/neurosnap/lists.sh/internal"
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
		if filepath.Ext(path) != ".txt" {
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

func syncRequest(body api.SyncRequest) error {
	url := fmt.Sprintf("%s/sync", viper.GetString("url"))
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(body)
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

    data := &map[string]string{}
    json.NewDecoder(resp.Body).Decode(data)
    fmt.Println(data)

	return nil
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

		syncReq := api.SyncRequest{}
		for _, file := range files {
            text, err := os.ReadFile(file)
            if err != nil {
                return err
            }
			req := api.PostRequest{
				Title: internal.FsToTitle(dir, file),
				Text:  string(text),
			}
			syncReq.Posts = append(syncReq.Posts, req)
		}

		fmt.Println(syncReq)
        syncRequest(syncReq)

		return nil
	},
}
