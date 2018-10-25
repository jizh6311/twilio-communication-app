package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "tce",
	Short: "TrafficClaim Enforcer",
	Long:  "Run the TrafficClaim Enforcer to manage Istio policy interactions",
}

func Execute() {
	fmt.Println("Execute")
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
}
