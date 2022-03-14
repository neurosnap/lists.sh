package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type LoginResponse struct {
	Id string `json:"id"`
}

type VerifyResponse struct {
	Token string `json:"token"`
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringP("email", "e", "", "email to login with")

	err := viper.BindPFlags(loginCmd.Flags())
	if err != nil {
		log.Println("Unable to bind pflags:", err)
	}
}

func loginRequest() (string, error) {
	url := fmt.Sprintf("%s/api/login", viper.GetString("url"))
	b := new(bytes.Buffer)
	m := map[string]string{"email": viper.GetString("email")}
	json.NewEncoder(b).Encode(m)
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data := &LoginResponse{}
	json.NewDecoder(resp.Body).Decode(data)

	return data.Id, nil
}

func verifyRequest(verificationId string, code string) (string, error) {
	url := fmt.Sprintf("%s/api/verify", viper.GetString("url"))
	b := new(bytes.Buffer)
	m := map[string]string{"id": verificationId, "code": code}
	json.NewEncoder(b).Encode(m)
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	data := &VerifyResponse{}
	json.NewDecoder(resp.Body).Decode(data)

	return data.Token, nil
}

func ask(text string) (string, error) {
	buf := bufio.NewReader(os.Stdin)
    fmt.Printf(text)
    res, err := buf.ReadString('\n')
    if err != nil {
        return "", err
    }

    return strings.ReplaceAll(res, "\n", ""), nil
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "authenticate with lists.sh",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Logging in with [%s] ...\n", viper.GetString("email"))
		verificationId, err := loginRequest()
		if err != nil {
			return err
		}

        fmt.Println("An email has been sent with a verification code")
        code, err := ask("Enter verification code: ")
        if err != nil {
			return err
		}

		token, err := verifyRequest(verificationId, code)
		if err != nil {
			return err
		}

		fmt.Println(token)

		return nil
	},
}
