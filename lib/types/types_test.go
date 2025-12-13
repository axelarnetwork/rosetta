package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/rosetta/lib/types"
)

func TestParseBalanceType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    types.BalanceType
		expectError bool
	}{
		{"available_balance", "available_balance", types.AvailableBalance, false},
		{"pending_rewards", "pending_rewards", types.PendingRewards, false},
		{"unbonding_balance", "unbonding_balance", types.UnbondingBalance, false},
		{"delegated_balance", "delegated_balance", types.DelegatedBalance, false},
		{"case insensitive", "DELEGATED_BALANCE", types.DelegatedBalance, false},
		{"mixed case", "Pending_Rewards", types.PendingRewards, false},
		{"unrecognized", "invalid_type", types.Unrecognized, true},
		{"empty", "", types.Unrecognized, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := types.ParseBalanceType(tt.input)
			if tt.expectError {
				require.Error(t, err)
				require.Equal(t, types.Unrecognized, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBalanceTypeString(t *testing.T) {
	tests := []struct {
		balanceType types.BalanceType
		expected    string
	}{
		{types.AvailableBalance, "available_balance"},
		{types.PendingRewards, "pending_rewards"},
		{types.UnbondingBalance, "unbonding_balance"},
		{types.DelegatedBalance, "delegated_balance"},
		{types.Unrecognized, "unrecognized_balance_type"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.balanceType.String())
		})
	}
}

func TestBalanceMetaData(t *testing.T) {
	meta := types.BalanceMetaData(types.DelegatedBalance, "axelarvaloper1abc")

	require.Equal(t, "delegated_balance", meta["balance_type"])
	require.Equal(t, "axelarvaloper1abc", meta["validator_address"])
}

func TestUnbondingDelegationMetaData(t *testing.T) {
	completionTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	meta := types.UnbondingDelegationMetaData("axelarvaloper1xyz", completionTime)

	require.Equal(t, "unbonding_balance", meta["balance_type"])
	require.Equal(t, "axelarvaloper1xyz", meta["validator_address"])
	require.Contains(t, meta["completion_time"], "2024-01-15")
}
