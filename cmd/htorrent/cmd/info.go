package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	raddrFlag  = "raddr"
	magnetFlag = "magnet"
)

var (
	errMissingAPIPassword = errors.New("missing API password")
	errMissingAPIUsername = errors.New("missing API username")
)

var infoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"i"},
	Short:   "Get streamable URLs from the gateway's info endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		if strings.TrimSpace(viper.GetString(apiPasswordFlag)) == "" {
			return errMissingAPIPassword
		}

		if strings.TrimSpace(viper.GetString(apiUsernameFlag)) == "" {
			return errMissingAPIUsername
		}

		if strings.TrimSpace(viper.GetString(magnetFlag)) == "" {
			return errEmptyMagnetLink
		}

		hc := &http.Client{}

		u, err := url.Parse(viper.GetString(raddrFlag))
		if err != nil {
			return err
		}

		suffix, err := url.Parse("/info")
		if err != nil {
			return err
		}
		u = u.ResolveReference(suffix)

		q := u.Query()
		q.Set("magnet", viper.GetString(magnetFlag))
		u.RawQuery = q.Encode()

		log.Println(u.String())

		req, err := http.NewRequest(http.MethodGet, u.String(), http.NoBody)
		if err != nil {
			return err
		}
		req.SetBasicAuth(viper.GetString(apiUsernameFlag), viper.GetString(apiPasswordFlag))

		res, err := hc.Do(req)
		if err != nil {
			return err
		}
		if res.Body != nil {
			defer res.Body.Close()
		}
		if res.StatusCode != http.StatusOK {
			return errors.New(res.Status)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}

		fmt.Println(string(body))

		return nil
	},
}

func init() {
	infoCmd.PersistentFlags().String(apiUsernameFlag, "admin", "Username for the gateway")
	infoCmd.PersistentFlags().String(apiPasswordFlag, "", "Username or OIDC access token for the gateway")
	infoCmd.PersistentFlags().String(raddrFlag, "http://localhost:1337/", "Remote address")
	infoCmd.PersistentFlags().String(magnetFlag, "", "Magnet link to get info for")

	viper.AutomaticEnv()

	rootCmd.AddCommand(infoCmd)
}
