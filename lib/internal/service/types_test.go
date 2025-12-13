package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/rosetta/lib/internal/service"
)

func TestSubAccountMetaData_FromMetadata(t *testing.T) {
	t.Run("with validator address", func(t *testing.T) {
		meta := map[string]interface{}{
			"validator_address": "axelarvaloper1abc123",
		}

		subAccount := new(service.SubAccountMetaData)
		err := subAccount.FromMetadata(meta)

		require.NoError(t, err)
		require.Equal(t, "axelarvaloper1abc123", subAccount.ValidatorAddress)
	})

	t.Run("without validator address", func(t *testing.T) {
		meta := map[string]interface{}{}

		subAccount := new(service.SubAccountMetaData)
		err := subAccount.FromMetadata(meta)

		require.NoError(t, err)
		require.Equal(t, "", subAccount.ValidatorAddress)
	})

	t.Run("nil metadata", func(t *testing.T) {
		subAccount := new(service.SubAccountMetaData)
		err := subAccount.FromMetadata(nil)

		require.NoError(t, err)
		require.Equal(t, "", subAccount.ValidatorAddress)
	})

	t.Run("extra fields ignored", func(t *testing.T) {
		meta := map[string]interface{}{
			"validator_address": "axelarvaloper1xyz",
			"extra_field":       "ignored",
		}

		subAccount := new(service.SubAccountMetaData)
		err := subAccount.FromMetadata(meta)

		require.NoError(t, err)
		require.Equal(t, "axelarvaloper1xyz", subAccount.ValidatorAddress)
	})
}
