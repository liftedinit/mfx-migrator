package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/mfx-migrator/internal/chain"
	"github.com/liftedinit/mfx-migrator/internal/store"
)

// TODO: Tests

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Execute the MFX token migration associated with the given UUID.",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := LoadConfigFromCLI("migrate-uuid")
		slog.Debug("args", "config", config)
		if err := config.Validate(); err != nil {
			return err
		}

		migrateConfig := LoadMigrationConfigFromCLI()
		slog.Debug("args", "migrate-config", migrateConfig)
		if err := migrateConfig.Validate(); err != nil {
			return err
		}

		slog.Info("Loading state...", "uuid", config.UUID)
		item, err := store.LoadState(config.UUID)
		if err != nil {
			slog.Error("unable to load state", "error", err)
			return err
		}

		if err := verifyItemStatus(item); err != nil {
			return err
		}

		r := CreateRestClient(cmd.Context(), config.Url, config.Neighborhood)
		if err := AuthenticateRestClient(r, config.Username, config.Password); err != nil {
			return err
		}

		return migrate(r, item, migrateConfig)
	},
}

func init() {
	setupMigrateFlags()
	rootCmd.AddCommand(migrateCmd)
}

// verifyItemStatus verifies the status of the work item is valid for migration.
func verifyItemStatus(item *store.WorkItem) error {
	if !(item.Status == store.CLAIMED || item.Status == store.MIGRATING) {
		slog.Error("work item status not valid for migration", "uuid", item.UUID, "status", item.Status)
		return fmt.Errorf("work item status not valid for migration: %s, %s", item.UUID, item.Status)
	}
	return nil
}

// compareItems compares the local and remote work items to ensure they match.
func compareItems(item *store.WorkItem, remoteItem *store.WorkItem) error {
	if !item.Equal(*remoteItem) {
		slog.Error("local and remote work items do not match", "local", item, "remote", remoteItem)
		return fmt.Errorf("local and remote work items do not match: %s, %s", item.UUID, remoteItem.UUID)
	}
	return nil
}

// TODO: Support migrating multiple work items at once
func setupMigrateFlags() {
	migrateCmd.Flags().String("chainId", "", "Chain ID of the blockchain to migrate to")
	if err := viper.BindPFlag("chainId", migrateCmd.Flags().Lookup("chainId")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().String("address-prefix", "", "Address prefix of the blockchain to migrate to")
	if err := viper.BindPFlag("address-prefix", migrateCmd.Flags().Lookup("address-prefix")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().String("node-address", "", "Node address of the blockchain to migrate to")
	if err := viper.BindPFlag("node-address", migrateCmd.Flags().Lookup("node-address")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().String("keyring-backend", "", "Keyring backend to use")
	if err := viper.BindPFlag("keyring-backend", migrateCmd.Flags().Lookup("keyring-backend")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().String("bank-address", "", "Bank address to send tokens from")
	if err := viper.BindPFlag("bank-address", migrateCmd.Flags().Lookup("bank-address")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().String("chain-home", "", "Root directory of the chain configuration")
	if err := viper.BindPFlag("chain-home", migrateCmd.Flags().Lookup("chain-home")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().Int64("amount", 0, "Amount of tokens to migrate")
	if err := viper.BindPFlag("amount", migrateCmd.Flags().Lookup("amount")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().String("denom", "", "Denomination of the tokens to migrate")
	if err := viper.BindPFlag("denom", migrateCmd.Flags().Lookup("denom")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
	migrateCmd.Flags().String("uuid", "", "UUID of the work item to claim")
	if err := migrateCmd.MarkFlagRequired("uuid"); err != nil {
		slog.Error(ErrorMarkingFlagRequired, "error", err)
	}
	if err := viper.BindPFlag("migrate-uuid", migrateCmd.Flags().Lookup("uuid")); err != nil {
		slog.Error(ErrorBindingFlag, "error", err)
	}
}

// migrate migrates a work item to the Manifest Ledger.
func migrate(r *resty.Client, item *store.WorkItem, config MigrateConfig) error {
	slog.Info("Migrating work item...", "uuid", item.UUID)

	remoteItem, err := store.GetWorkItem(r, item.UUID)
	if err != nil {
		return err
	}

	// Verify the item is ready for migration
	if err = verifyItemStatus(remoteItem); err != nil {
		return err
	}

	// Verify the local and remote items match
	if err = compareItems(item, remoteItem); err != nil {
		return err
	}

	var newItem = *item

	// If the item status is not MIGRATING, set it to MIGRATING
	if newItem.Status != store.MIGRATING {
		if err = setAsMigrating(r, newItem); err != nil {
			return err
		}
	}

	// Send the tokens
	txHash, blockTime, err := sendTokens(r, &newItem, config)
	if err != nil {
		return err
	}

	slog.Info("Migration succeeded on chain...", "hash", txHash, "timestamp", blockTime)
	// Set the status to COMPLETED
	if err = setAsCompleted(r, newItem, txHash, blockTime); err != nil {
		return err
	}

	// Delete the state file, as the work item is now completed and the state is stored in the database
	if err = deleteState(&newItem); err != nil {
		return err
	}

	slog.Info("Migration complete", "uuid", newItem.UUID)

	return nil
}

func deleteState(item *store.WorkItem) error {
	slog.Info("Deleting local state file...")
	if err := os.Remove(fmt.Sprintf("%s.json", item.UUID)); err != nil {
		slog.Error("could not delete state", "error", err)
		return err
	}
	return nil
}

// setAsMigrating sets the status of the work item to MIGRATING and updates the state.
func setAsMigrating(r *resty.Client, newItem store.WorkItem) error {
	newItem.Status = store.MIGRATING
	if err := store.UpdateWorkItemAndSaveState(r, newItem); err != nil {
		return err
	}
	return nil
}

// setAsCompleted sets the status of the work item to COMPLETED.
// It also sets the manifest hash and updates the state.
func setAsCompleted(r *resty.Client, newItem store.WorkItem, txHash *string, blockTime *time.Time) error {
	newItem.Status = store.COMPLETED
	newItem.ManifestHash = txHash
	newItem.ManifestDatetime = blockTime
	if err := store.UpdateWorkItemAndSaveState(r, newItem); err != nil {
		return err
	}
	return nil
}

func setAsFailed(r *resty.Client, newItem store.WorkItem, errStr *string) error {
	newItem.Status = store.FAILED
	newItem.Error = errStr
	if err := store.UpdateWorkItemAndSaveState(r, newItem); err != nil {
		return err
	}
	return nil
}

// sendTokens sends the tokens from the bank account to the user account.
func sendTokens(r *resty.Client, item *store.WorkItem, config MigrateConfig) (*string, *time.Time, error) {
	txResponse, blockTime, err := chain.Migrate(item, chain.MigrationConfig{
		ChainID:        config.ChainID,
		NodeAddress:    config.NodeAddress,
		KeyringBackend: config.KeyringBackend,
		ChainHome:      config.ChainHome,
		AddressPrefix:  config.AddressPrefix,
		BankAddress:    config.BankAddress,
		Amount:         config.Amount,
		Denom:          config.Denom,
	})
	if err != nil {
		slog.Error("error during migration, operator intervention required", "error", err)
		errStr := err.Error()
		if fErr := setAsFailed(r, *item, &errStr); fErr != nil {
			return nil, nil, fErr
		}

		return nil, nil, err
	}

	if txResponse.Code != 0 {
		slog.Error("migration failed", "code", txResponse.Code, "log", txResponse.RawLog)
		return nil, nil, fmt.Errorf("migration failed: %s", txResponse.RawLog)
	}

	return &txResponse.TxHash, blockTime, nil
}
