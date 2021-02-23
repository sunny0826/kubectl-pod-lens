package cli

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sunny0826/kubectl-sniffer/pkg/plugin"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"strings"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "sniffer",
		Short:         "",
		Long:          `.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("A pod name is required!")
			}

			podName := args[0]
			argsChannel := make(chan string, 1)
			argsChannel <- podName

			if err := plugin.RunPlugin(KubernetesConfigFlags, argsChannel); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	KubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	KubernetesConfigFlags.AddFlags(cmd.Flags())

	cmd.Flags().MarkHidden("as-group")
	cmd.Flags().MarkHidden("as")
	cmd.Flags().MarkHidden("cache-dir")
	cmd.Flags().MarkHidden("certificate-authority")
	cmd.Flags().MarkHidden("client-certificate")
	cmd.Flags().MarkHidden("client-key")
	cmd.Flags().MarkHidden("cluster")
	cmd.Flags().MarkHidden("context")
	cmd.Flags().MarkHidden("insecure-skip-tls-verify")
	cmd.Flags().MarkHidden("kubeconfig")
	cmd.Flags().MarkHidden("password")
	cmd.Flags().MarkHidden("request-timeout")
	cmd.Flags().MarkHidden("server")
	cmd.Flags().MarkHidden("token")
	cmd.Flags().MarkHidden("user")
	cmd.Flags().MarkHidden("username")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
}
