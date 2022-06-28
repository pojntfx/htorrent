package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/pojntfx/htorrent/pkg/client"
	"github.com/pojntfx/htorrent/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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

type infoWithStreamURL struct {
	Name         string              `yaml:"name"`
	Description  string              `yaml:"description"`
	CreationDate int64               `yaml:"creationDate"`
	Files        []fileWithStreamURL `yaml:"files"`
}

type fileWithStreamURL struct {
	Path      string `yaml:"path"`
	Length    int64  `yaml:"length"`
	StreamURL string `yaml:"streamURL"`
}

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

		info, err := manager.GetInfo(viper.GetString(magnetFlag))
		if err != nil {
			return err
		}

		if strings.TrimSpace(viper.GetString(expressionFlag)) == "" {
			i := infoWithStreamURL{
				Name:         info.Name,
				Description:  info.Description,
				CreationDate: info.CreationDate,
				Files:        []fileWithStreamURL{},
			}

			for _, f := range info.Files {
				streamURL, err := getStreamURL(viper.GetString(raddrFlag), viper.GetString(magnetFlag), f.Path)
				if err != nil {
					return err
				}

				i.Files = append(i.Files, fileWithStreamURL{
					Path:      f.Path,
					Length:    f.Length,
					StreamURL: streamURL,
				})
			}

			y, err := yaml.Marshal(i)
			if err != nil {
				return err
			}

			fmt.Printf("%s", y)
		} else {
			exp := regexp.MustCompile(viper.GetString(expressionFlag))

			for _, f := range info.Files {
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
