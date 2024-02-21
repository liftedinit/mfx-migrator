package cmd

import (
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/mfx-migrator/internal/state"
)

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify the status of a migration of MFX tokens to the Manifest Ledger",
	RunE: func(cmd *cobra.Command, args []string) error {
		url := viper.GetString("url")
		uuidStr := viper.GetString("uuid")
		if uuidStr == "" {
			slog.Error("uuid is required")
			return errors.New("uuid is required")
		}
		workItemUUID := uuid.MustParse(uuidStr)

		s, err := state.LoadState(workItemUUID)
		if err != nil {
			slog.Warn("unable to load local state, continuing", "error", err)
		}

		if s != nil {
			slog.Debug("local state loaded", "state", s)
		}

		// Verify the work item on the remote database
		slog.Debug("verifying remote state", "url", url, "uuid", uuidStr)

		return nil
	},
}

func init() {
	verifyCmd.Flags().StringP("uuid", "u", "", "UUID of the MFX migration")
	err := viper.BindPFlag("uuid", verifyCmd.Flags().Lookup("uuid"))
	if err != nil {
		slog.Error("unable to bind flag", "error", err)
	}

	rootCmd.AddCommand(verifyCmd)
}
