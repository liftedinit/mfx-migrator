package interchaintest

import (
	sdkmath "cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

const (
	votingPeriod     = "15s"
	maxDepositPeriod = "10s"
	Denom            = "umfx"

	accAddr        = "manifest1hj5fveer5cjtn4wd6wstzugjfdxzl0xp8ws9ct" // POA
	gasStationAddr = "manifest1efd63aw40lxf3n4mhf7dzhjkr453axurm6rp3z" // Gas Station
	bankAddr       = "manifest1atjlsfg8s2hpvszn5dr8z7d3usvs70tg3ssa84" // Bank

	// manifest1atjlsfg8s2hpvszn5dr8z7d3usvs70tg3ssa84
	userMnemonic = "tuna develop gap truly crew canoe enlist slim stove scorpion clerk absurd better surprise moon fiction bean poem car air proud prevent unknown glue"

	// manifest1efd63aw40lxf3n4mhf7dzhjkr453axurm6rp3z
	user2Mnemonic   = "wealth flavor believe regret funny network recall kiss grape useless pepper cram hint member few certain unveil rather brick bargain curious require crowd raise"
	chainType       = "cosmos"
	chainName       = "manifest-ledger"
	chainID         = "manifest-2"
	chainRepository = "ghcr.io/liftedinit/manifest-ledger"
	chainVersion    = "v1.0.3"
)

var (
	vals           = 1
	fullNodes      = 0
	DefaultGenesis = []cosmos.GenesisKV{
		// Governance
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", votingPeriod),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", maxDepositPeriod),
		// ABCI++
		cosmos.NewGenesisKV("consensus.params.abci.vote_extensions_enable_height", "1"),
		// TokenFactory
		cosmos.NewGenesisKV("app_state.tokenfactory.params.denom_creation_fee", nil),
		cosmos.NewGenesisKV("app_state.tokenfactory.params.denom_creation_gas_consume", "1"),
		// Mint - this is the only param the manifest module depends on from mint
		cosmos.NewGenesisKV("app_state.mint.params.blocks_per_year", "6311520"),
		// FeeGrant
		cosmos.NewGenesisKV("app_state.feegrant.allowances", []interface{}{
			map[string]interface{}{
				"granter": gasStationAddr,
				"grantee": bankAddr,
				"allowance": map[string]interface{}{
					"@type": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
					"allowance": map[string]interface{}{
						"@type":       "/cosmos.feegrant.v1beta1.BasicAllowance",
						"spend_limit": []interface{}{},
						"expiration":  nil,
					},
					"allowed_messages": []string{
						"/cosmos.bank.v1beta1.MsgSend",
					},
				},
			},
		}),
	}

	LocalChainConfig = ibc.ChainConfig{
		Type:    chainType,
		Name:    chainName,
		ChainID: chainID,
		Images: []ibc.DockerImage{
			{
				Repository: chainRepository,
				Version:    chainVersion,
				UidGid:     "1025:1025",
			},
		},
		Bin:            "manifestd",
		Bech32Prefix:   "manifest",
		Denom:          Denom,
		GasPrices:      "0.0011" + Denom,
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
		EncodingConfig: AppEncoding(),
		ModifyGenesis:  cosmos.ModifyGenesis(DefaultGenesis),
	}

	DefaultGenesisAmt = sdkmath.NewInt(10_000_000_000)
)
