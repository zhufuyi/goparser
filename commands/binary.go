package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/zhufuyi/goparser/parser"
)

// parse go binary command
func parseGoBinaryCMD() *cobra.Command {
	var (
		binaryFile string // binary file path
		topN       int    // show top N information
		grep       string // grep symbol name
		sortName   string // info sort, size, address, or symbol
		isAsc      bool   // sort order, true: asc, false: desc
		maxWidth   int    // max width of output
	)

	cmd := &cobra.Command{
		Use:   "binary",
		Short: "Parse binary file compiled by go",
		Long:  "Parse binary file compiled by go.",
		Example: color.HiBlackString(`  # Parse the binary file compiled by go
  goparser binary --binary-file=./your_binary_file

  # Parse the binary file compiled by go and show top 30 information
  goparser binary --binary-file=./your_binary_file --top-n=30

  # Parse the binary file compiled by go and grep symbol name "sponge"
  goparser binary --binary-file=./your_binary_file --grep=sponge`),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println("Error:", checkErr(e.(error)))
				}
			}()

			if maxWidth < 50 {
				maxWidth = 50
			} else if maxWidth > 256 {
				maxWidth = 256
			}

			bp, err := parser.NewBinaryParser(binaryFile, grep)
			if err != nil {
				panic(err)
			}
			bp.MaxWidth = maxWidth

			sortName = strings.ToLower(sortName)
			switch sortName {
			case "address", "addr":
				sort.Sort(parser.ByNmAddress{NmParsers: bp.NmParsers, IsAsc: isAsc})
			case "symbol", "sym":
				sort.Sort(parser.ByNmSymbol{NmParsers: bp.NmParsers, IsAsc: isAsc})
			default:
				sort.Sort(parser.ByNmSize{NmParsers: bp.NmParsers, IsAsc: isAsc})
			}
			sort.Sort(parser.ByPkgSize{PkgInfos: bp.PkgInfos})

			bp.PrintNmParser(binaryFile, topN)
			fmt.Printf("\n\n")
			bp.PrintPkgInfo(binaryFile, topN)

			return nil
		},
	}

	cmd.Flags().StringVarP(&binaryFile, "binary-file", "f", "", "binary file path")
	_ = cmd.MarkFlagRequired("binary-file")
	cmd.Flags().IntVarP(&topN, "top-n", "n", 100, "show top N information")
	cmd.Flags().StringVarP(&grep, "grep", "g", "", "grep symbol name")
	cmd.Flags().StringVarP(&sortName, "sort", "s", "size", "info sort, size, address, or symbol")
	cmd.Flags().BoolVarP(&isAsc, "asc", "a", false, "sort order, true: asc, false: desc")
	cmd.Flags().IntVarP(&maxWidth, "max-width", "w", 60, "max width of output")

	return cmd
}

func checkErr(err error) error {
	if strings.Contains(err.Error(), "no symbols") {
		paramTip := color.HiRedString(`-ldflags "-s -w"`)
		cmdTip := color.HiCyanString("go build")
		return fmt.Errorf("there is no symbol information in the compiled binary file, please do not use the parameter %s for the %s command.", paramTip, cmdTip)
	}
	return err
}
