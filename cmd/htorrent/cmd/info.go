package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	raddrFlag      = "raddr"
	magnetFlag     = "magnet"
	expressionFlag = "expression"
)

var (
	errMissingAPIPassword      = errors.New("missing API password")
	errMissingAPIUsername      = errors.New("missing API username")
	errNoPathMatchesExpression = errors.New("could not find a path that matches the supplied expression")
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

		baseURL, err := url.Parse(viper.GetString(raddrFlag))
		if err != nil {
			return err
		}

		infoSuffix, err := url.Parse("/info")
		if err != nil {
			return err
		}

		info := baseURL.ResolveReference(infoSuffix)

		q := info.Query()
		q.Set("magnet", viper.GetString(magnetFlag))
		info.RawQuery = q.Encode()

		req, err := http.NewRequest(http.MethodGet, info.String(), http.NoBody)
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

		files := []file{}
		dec := json.NewDecoder(res.Body)
		if err := dec.Decode(&files); err != nil {
			return err
		}

		if strings.TrimSpace(viper.GetString(expressionFlag)) == "" {
			w := csv.NewWriter(os.Stdout)
			defer w.Flush()

			if err := w.Write([]string{"path", "length", "creationTime", "streamURL"}); err != nil {
				return err
			}

			for _, f := range files {
				streamURL, err := getStreamURL(baseURL, viper.GetString(magnetFlag), f.Path)
				if err != nil {
					return err
				}

				if err := w.Write([]string{f.Path, fmt.Sprintf("%v", f.Length), time.Unix(f.CreationDate, 0).Format(time.RFC3339), streamURL}); err != nil {
					return err
				}
			}
		} else {
			exp := regexp.MustCompile(viper.GetString(expressionFlag))

			for _, f := range files {
				if exp.Match([]byte(f.Path)) {
					streamURL, err := getStreamURL(baseURL, viper.GetString(magnetFlag), f.Path)
					if err != nil {
						return err
					}

					fmt.Println(streamURL)

					return nil
				}
			}

			return errNoPathMatchesExpression
		}

		return nil
	},
}

func getStreamURL(base *url.URL, magnet, path string) (string, error) {
	streamSuffix, err := url.Parse("/stream")
	if err != nil {
		return "", err
	}

	stream := base.ResolveReference(streamSuffix)

	q := stream.Query()
	q.Set("magnet", magnet)
	q.Set("path", path)
	stream.RawQuery = q.Encode()

	return stream.String(), nil
}

func init() {
	infoCmd.PersistentFlags().StringP(apiUsernameFlag, "u", "admin", "Username for the gateway")
	infoCmd.PersistentFlags().StringP(apiPasswordFlag, "p", "", "Username or OIDC access token for the gateway")
	infoCmd.PersistentFlags().StringP(raddrFlag, "r", "http://localhost:1337/", "Remote address")
	infoCmd.PersistentFlags().StringP(magnetFlag, "m", "", "Magnet link to get info for")
	infoCmd.PersistentFlags().StringP(expressionFlag, "x", "", "Regex to select the link to output by, i.e. (.*).mkv$ to only return the first .mkv file; disables all other info")

	viper.AutomaticEnv()

	rootCmd.AddCommand(infoCmd)
}
