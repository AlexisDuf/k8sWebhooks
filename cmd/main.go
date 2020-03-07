package main

import (
	"flag"

	webhooks "github.com/AlexisDuf/k8sWebhooks/pkg/server"
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

func main() {
	rootCmd := &cobra.Command{Use: "app", Version: "2.12"}

	rootCmd.AddCommand(webhooks.CmdServer)

	// NOTE(claudiub): Some tests are passing logging related flags, so we need to be able to
	// accept them. This will also include them in the printed help.
	loggingFlags := &flag.FlagSet{}
	klog.InitFlags(loggingFlags)
	rootCmd.PersistentFlags().AddGoFlagSet(loggingFlags)
	rootCmd.Execute()
}
