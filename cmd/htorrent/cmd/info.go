package cmd

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pojntfx/htorrent/pkg/client"
	"github.com/pojntfx/htorrent/pkg/server"
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
			return server.ErrEmptyMagnetLink
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := client.NewManager(
			viper.GetString(raddrFlag),
			viper.GetString(apiUsernameFlag),
			viper.GetString(apiPasswordFlag),
			ctx,
		)

		files, err := manager.GetInfo(viper.GetString(magnetFlag))
		if err != nil {
			return err
		}

		if strings.TrimSpace(viper.GetString(expressionFlag)) == "" {
			w := csv.NewWriter(os.Stdout)
			defer w.Flush()

			if err := w.Write([]string{"path", "length", "creationTime", "streamURL"}); err != nil {
				return err
			}

			for _, f := range files {
				streamURL, err := getStreamURL(viper.GetString(raddrFlag), viper.GetString(magnetFlag), f.Path)
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
					streamURL, err := getStreamURL(viper.GetString(raddrFlag), viper.GetString(magnetFlag), f.Path)
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

func getStreamURL(base string, magnet, path string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	streamSuffix, err := url.Parse("/stream")
	if err != nil {
		return "", err
	}

	stream := baseURL.ResolveReference(streamSuffix)

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
