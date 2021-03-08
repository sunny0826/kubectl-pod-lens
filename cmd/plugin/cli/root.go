package cli

import (
	"fmt"
	"os"
	"strings"

	"k8s.io/klog"

	"github.com/i582/cfmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sunny0826/kubectl-pod-lens/pkg/plugin"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubectl pod-lens [pod name]",
		Short: "View pod related resources.",
		Long:  printLogo(),
		Example: `
# Sniffing pod-related resources
$ kubectl pod-lens prometheus-prometheus-operator-prometheus-0 
`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
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

	klog.InitFlags(nil)
	_ = cmd.Flags().MarkHidden("as-group")
	_ = cmd.Flags().MarkHidden("as")
	_ = cmd.Flags().MarkHidden("cache-dir")
	_ = cmd.Flags().MarkHidden("certificate-authority")
	_ = cmd.Flags().MarkHidden("client-certificate")
	_ = cmd.Flags().MarkHidden("client-key")
	_ = cmd.Flags().MarkHidden("cluster")
	_ = cmd.Flags().MarkHidden("insecure-skip-tls-verify")
	_ = cmd.Flags().MarkHidden("kubeconfig")
	_ = cmd.Flags().MarkHidden("password")
	_ = cmd.Flags().MarkHidden("request-timeout")
	_ = cmd.Flags().MarkHidden("server")
	_ = cmd.Flags().MarkHidden("token")
	_ = cmd.Flags().MarkHidden("user")
	_ = cmd.Flags().MarkHidden("username")

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

func printLogo() string {
	return cfmt.Sprintf(`
{{                           /$$         /$$                               }}::red
{{                          | $$        | $$                               }}::red
{{  /$$$$$$   /$$$$$$   /$$$$$$$        | $$  /$$$$$$  /$$$$$$$   /$$$$$$$ }}::yellow
{{ /$$__  $$ /$$__  $$ /$$__  $$ /$$$$$$| $$ /$$__  $$| $$__  $$ /$$_____/ }}::yellow
{{| $$  \ $$| $$  \ $$| $$  | $$|______/| $$| $$$$$$$$| $$  \ $$|  $$$$$$  }}::blue
{{| $$  | $$| $$  | $$| $$  | $$        | $$| $$_____/| $$  | $$ \____  $$ }}::blue
{{| $$$$$$$/|  $$$$$$/|  $$$$$$$        | $$|  $$$$$$$| $$  | $$ /$$$$$$$/ }}::green
{{| $$____/  \______/  \_______/        |__/ \_______/|__/  |__/|_______/  }}::green
{{| $$                                                                     }}::white
{{| $$                                                                     }}::white
{{|__/                                                                     }}::white

Find related {{workloads}}::green|underline, {{namespace}}::green|underline, {{node}}::green|underline, {{service}}::green|underline, {{configmap}}::green|underline, {{secret}}::green|underline, {{ingress}}::green|underline and {{HPA}}::green|underline 
by {{pod name}}::lightRed and display them in a {{tree}}::blue and {{table}}::blue.
Find more information at: {{https://github.com/sunny0826/kubectl-sniffer}}::lightMagenta|underline
`)
}
