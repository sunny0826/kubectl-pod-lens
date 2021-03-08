package main

import (
	"github.com/sunny0826/kubectl-pod-lens/cmd/plugin/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // required for GKE
	"k8s.io/klog"
)

func main() {
	defer klog.Flush()
	cli.InitAndExecute()
}
