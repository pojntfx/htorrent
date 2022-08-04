package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/pojntfx/htorrent/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var metricsCmd = &cobra.Command{
	Use:     "metrics",
	Aliases: []string{"m"},
	Short:   "Get metrics from the gateway",
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager := client.NewManager(
			viper.GetString(raddrFlag),
			viper.GetString(apiUsernameFlag),
			viper.GetString(apiPasswordFlag),
			ctx,
		)

		metrics, err := manager.GetMetrics()
		if err != nil {
			return err
		}

		y, err := yaml.Marshal(metrics)
		if err != nil {
			return err
		}

		fmt.Printf("%s", y)

		return nil
	},
}

func init() {
	metricsCmd.PersistentFlags().StringP(apiUsernameFlag, "u", "admin", "Username for the gateway")
	metricsCmd.PersistentFlags().StringP(apiPasswordFlag, "p", "", "Username or OIDC access token for the gateway")
	metricsCmd.PersistentFlags().StringP(raddrFlag, "r", "http://localhost:1337/", "Remote address")

	viper.AutomaticEnv()

	rootCmd.AddCommand(metricsCmd)
}
