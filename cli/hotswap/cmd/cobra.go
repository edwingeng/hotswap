package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hotswap",
	Short: "Build hotswap-able golang plugin",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
