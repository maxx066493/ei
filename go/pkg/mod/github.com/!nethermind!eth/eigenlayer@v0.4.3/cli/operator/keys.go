package operator

import (
	"github.com/NethermindEth/eigenlayer/cli/operator/keys"
	"github.com/NethermindEth/eigenlayer/cli/prompter"
	"github.com/spf13/cobra"
)

func KeysCmd(p prompter.Prompter) *cobra.Command {
	cmd := cobra.Command{
		Use:   "keys",
		Short: "Manage the operator's keys",
	}

	cmd.AddCommand(
		keys.CreateCmd(p),
		keys.ListCmd(p),
		keys.ImportCmd(p),
	)

	return &cmd
}
