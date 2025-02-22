package commands

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewRootCMD command entry
func NewRootCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "goparser",
		Long:          color.HiBlackString(`goparser is a parse tool for Golang source code.`),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(
		parseGoModCMD(),
		parseGoBinaryCMD(),
	)

	return cmd
}
