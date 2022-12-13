package cmd

import (
	"errors"
	"fmt"
	"github.com/open-policy-agent/opa/misc/regopls"
	"github.com/spf13/cobra"
	"os"
)

type regoplsCommandParams struct {
	testParam	bool
}

var regoplsParams = regoplsCommandParams{}

var regoplsCommand = &cobra.Command{
	Use:	"regopls [port]",
	Short: 	"Start Rego language server",
	Long:	`Start Rego language server.
WIP.`,
	Run: func(cmd *cobra.Command, args []string) {
		_, err := startLanguageServer(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error %v\n", err)
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	},
}

func startLanguageServer(args []string) (int, error) {
	if len(args) != 1 {
		return 0, errors.New("incorrect number of args")
	}
	regopls.Regopls()
	return 0, nil
}

func init() {
	regoplsCommand.Flags().BoolVarP(&regoplsParams.testParam, "test", "t", false, "test parameter")
	RootCommand.AddCommand(regoplsCommand)
}
