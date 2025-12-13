package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/cosmos/rosetta/lib/errors"
	crgtypes "github.com/cosmos/rosetta/lib/types"
)

// AccountBalance retrieves the account balance of an address
// rosetta requires us to fetch the block information too
func (on OnlineNetwork) AccountBalance(ctx context.Context, request *types.AccountBalanceRequest) (*types.AccountBalanceResponse, *types.Error) {
	var (
		height int64
		block  crgtypes.BlockResponse
		err    error
	)

	switch {
	case request.BlockIdentifier == nil:
		syncStatus, err := on.client.Status(ctx)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
		block, err = on.client.BlockByHeight(ctx, syncStatus.CurrentIndex)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
	case request.BlockIdentifier.Hash != nil:
		block, err = on.client.BlockByHash(ctx, *request.BlockIdentifier.Hash)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
	case request.BlockIdentifier.Index != nil:
		height = *request.BlockIdentifier.Index
		block, err = on.client.BlockByHeight(ctx, &height)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
	}

	height = block.Block.Index

	// determine balance type from sub-account
	balanceType := crgtypes.AvailableBalance
	var validatorAddress string
	if request.AccountIdentifier.SubAccount != nil {
		subAccount := request.AccountIdentifier.SubAccount
		balanceType, err = crgtypes.ParseBalanceType(subAccount.Address)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}

		metaData := new(SubAccountMetaData)
		err = metaData.FromMetadata(subAccount.Metadata)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
		validatorAddress = metaData.ValidatorAddress
	}

	var accountCoins []*types.Amount
	switch balanceType {
	case crgtypes.AvailableBalance:
		accountCoins, err = on.client.Balances(ctx, request.AccountIdentifier.Address, &height)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
	case crgtypes.DelegatedBalance:
		accountCoins, err = on.client.Delegations(ctx, request.AccountIdentifier.Address, &height)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
	case crgtypes.UnbondingBalance:
		accountCoins, err = on.client.UnbondingDelegations(ctx, request.AccountIdentifier.Address, &height)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
	case crgtypes.PendingRewards:
		accountCoins, err = on.client.Rewards(ctx, request.AccountIdentifier.Address, validatorAddress, &height)
		if err != nil {
			return nil, errors.ToRosetta(err)
		}
	default:
		err = errors.WrapError(errors.ErrBadArgument, "unrecognized balance type")
		return nil, errors.ToRosetta(err)
	}

	// fetch account sequence number, default to 0 if account doesn't exist
	sequence, err := on.client.AccountSequence(ctx, request.AccountIdentifier.Address, &height)
	if err != nil {
		sequence = 0
	}

	return &types.AccountBalanceResponse{
		BlockIdentifier: block.Block,
		Balances:        accountCoins,
		Metadata: map[string]interface{}{
			"sequence_number": sequence,
		},
	}, nil
}

// AccountsCoins - relevant only for UTXO based chain
// see https://www.rosetta-api.org/docs/AccountApi.html#accountcoins
func (on OnlineNetwork) AccountCoins(_ context.Context, _ *types.AccountCoinsRequest) (*types.AccountCoinsResponse, *types.Error) {
	return nil, errors.ToRosetta(errors.ErrOffline)
}
