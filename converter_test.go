package rosetta_test

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	rosettatypes "github.com/coinbase/rosetta-sdk-go/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/rosetta"
	crgerrs "github.com/cosmos/rosetta/lib/errors"
)

type ConverterTestSuite struct {
	suite.Suite

	c               rosetta.Converter
	unsignedTxBytes []byte
	unsignedTx      authsigning.Tx

	ir     codectypes.InterfaceRegistry
	cdc    *codec.ProtoCodec
	txConf client.TxConfig
}

func (s *ConverterTestSuite) SetupTest() {
	// create an unsigned tx
	const unsignedTxHex = "0a8e010a8b010a1c2f636f736d6f732e62616e6b2e763162657461312e4d736753656e64126b0a2d636f736d6f733134376b6c68377468356a6b6a793361616a736a3272717668747668396d666465333777713567122d636f736d6f73316d6e7670386c786b616679346c787777617175356561653764787630647a36687767797436331a0b0a057374616b651202313612600a4c0a460a1f2f636f736d6f732e63727970746f2e736563703235366b312e5075624b657912230a21034c92046950c876f4a5cb6c7797d6eeb9ef80d67ced4d45fb62b1e859240ba9ad12020a0012100a0a0a057374616b651201311090a10f1a00"
	unsignedTxBytes, err := hex.DecodeString(unsignedTxHex)
	s.Require().NoError(err)
	s.unsignedTxBytes = unsignedTxBytes
	// instantiate converter
	cdc, ir := rosetta.MakeCodec()
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	s.c = rosetta.NewConverter(cdc, ir, txConfig, address.NewBech32Codec("cosmos"), nil)
	// add utils
	s.ir = ir
	s.cdc = cdc
	s.txConf = txConfig
	// add authsigning tx
	sdkTx, err := txConfig.TxDecoder()(unsignedTxBytes)
	s.Require().NoError(err)
	builder, err := txConfig.WrapTxBuilder(sdkTx)
	s.Require().NoError(err)

	s.unsignedTx = builder.GetTx()
}

func (s *ConverterTestSuite) TestFromRosettaOpsToTxSuccess() {
	addr1 := sdk.AccAddress("address1").String()
	addr2 := sdk.AccAddress("address2").String()

	msg1 := &bank.MsgSend{
		FromAddress: addr1,
		ToAddress:   addr2,
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("test", 10)),
	}

	msg2 := &bank.MsgSend{
		FromAddress: addr2,
		ToAddress:   addr1,
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("utxo", 10)),
	}

	ops, err := s.c.ToRosetta().Ops("", msg1)
	s.Require().NoError(err)

	ops2, err := s.c.ToRosetta().Ops("", msg2)
	s.Require().NoError(err)

	ops = append(ops, ops2...)

	tx, err := s.c.ToSDK().UnsignedTx(ops)
	s.Require().NoError(err)

	getMsgs := tx.GetMsgs()

	s.Require().Equal(2, len(getMsgs))

	s.Require().Equal(getMsgs[0], msg1)
	s.Require().Equal(getMsgs[1], msg2)
}

func (s *ConverterTestSuite) TestFromRosettaOpsToTxErrors() {
	s.Run("unrecognized op", func() {
		op := &rosettatypes.Operation{
			Type: "non-existent",
		}

		_, err := s.c.ToSDK().UnsignedTx([]*rosettatypes.Operation{op})

		s.Require().ErrorIs(err, crgerrs.ErrBadArgument)
	})

	s.Run("codec type but not sdk.Msg", func() {
		op := &rosettatypes.Operation{
			Type: "cosmos.crypto.ed25519.PubKey",
		}

		_, err := s.c.ToSDK().UnsignedTx([]*rosettatypes.Operation{op})

		s.Require().ErrorIs(err, crgerrs.ErrBadArgument)
	})
}

func (s *ConverterTestSuite) TestMsgToMetaMetaToMsg() {
	msg := &bank.MsgSend{
		FromAddress: "addr1",
		ToAddress:   "addr2",
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("test", 10)),
	}

	meta, err := s.c.ToRosetta().Meta(msg)
	s.Require().NoError(err)

	copyMsg := new(bank.MsgSend)
	err = s.c.ToSDK().Msg(meta, copyMsg)
	s.Require().NoError(err)
	s.Require().Equal(msg, copyMsg)
}

func (s *ConverterTestSuite) TestSignedTx() {
	s.Run("success", func() {
		const payloadsJSON = `[{"hex_bytes":"82ccce81a3e4a7272249f0e25c3037a316ee2acce76eb0c25db00ef6634a4d57303b2420edfdb4c9a635ad8851fe5c7a9379b7bc2baadc7d74f7e76ac97459b5","signing_payload":{"address":"cosmos147klh7th5jkjy3aajsj2rqvhtvh9mfde37wq5g","hex_bytes":"ed574d84b095250280de38bf8c254e4a1f8755e5bd300b1f6ca2671688136ecc","account_identifier":{"address":"cosmos147klh7th5jkjy3aajsj2rqvhtvh9mfde37wq5g"},"signature_type":"ecdsa"},"public_key":{"hex_bytes":"034c92046950c876f4a5cb6c7797d6eeb9ef80d67ced4d45fb62b1e859240ba9ad","curve_type":"secp256k1"},"signature_type":"ecdsa"}]`
		const expectedSignedTxHex = "0a8e010a8b010a1c2f636f736d6f732e62616e6b2e763162657461312e4d736753656e64126b0a2d636f736d6f733134376b6c68377468356a6b6a793361616a736a3272717668747668396d666465333777713567122d636f736d6f73316d6e7670386c786b616679346c787777617175356561653764787630647a36687767797436331a0b0a057374616b651202313612620a4e0a460a1f2f636f736d6f732e63727970746f2e736563703235366b312e5075624b657912230a21034c92046950c876f4a5cb6c7797d6eeb9ef80d67ced4d45fb62b1e859240ba9ad12040a02087f12100a0a0a057374616b651201311090a10f1a4082ccce81a3e4a7272249f0e25c3037a316ee2acce76eb0c25db00ef6634a4d57303b2420edfdb4c9a635ad8851fe5c7a9379b7bc2baadc7d74f7e76ac97459b5"

		var payloads []*rosettatypes.Signature
		s.Require().NoError(json.Unmarshal([]byte(payloadsJSON), &payloads))

		signedTx, err := s.c.ToSDK().SignedTx(s.unsignedTxBytes, payloads)
		s.Require().NoError(err)

		signedTxHex := hex.EncodeToString(signedTx)

		s.Require().Equal(signedTxHex, expectedSignedTxHex)
	})

	s.Run("signers data and signing payloads mismatch", func() {
		_, err := s.c.ToSDK().SignedTx(s.unsignedTxBytes, nil)
		s.Require().ErrorIs(err, crgerrs.ErrInvalidTransaction)
	})
}

func (s *ConverterTestSuite) TestOpsAndSigners() {
	s.Run("success", func() {
		addr1 := sdk.AccAddress("address1")
		addr2 := sdk.AccAddress("address2")

		msg := &bank.MsgSend{
			FromAddress: addr1.String(),
			ToAddress:   addr2.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("test", 10)),
		}

		builder := s.txConf.NewTxBuilder()
		s.Require().NoError(builder.SetMsgs(msg))

		sdkTx := builder.GetTx()
		txBytes, err := s.txConf.TxEncoder()(sdkTx)
		s.Require().NoError(err)

		ops, signers, err := s.c.ToRosetta().OpsAndSigners(txBytes)
		s.Require().NoError(err)

		s.Require().Equal(1, len(ops), "should have one operation")
		s.Require().Equal(rosetta.TransferOperation, ops[0].Type)
		s.Require().Equal(addr1.String(), ops[0].Account.Address)

		s.Require().Equal(1, len(signers), "should have one signer")
		s.Require().Equal(addr1.String(), signers[0].Address, "signer address should match sender bech32 address")
	})
}

func (s *ConverterTestSuite) TestFinalizeBlockHashToTxType() {
	const deliverTxHex = "5229A67AA008B5C5F1A0AEA77D4DEBE146297A30AAEF01777AF10FAD62DD36AB"

	deliverTxBytes, err := hex.DecodeString(deliverTxHex)
	s.Require().NoError(err)

	finalizeBlockTxHex := s.c.ToRosetta().FinalizeBlockTxHash(deliverTxBytes)

	txType, hash := s.c.ToSDK().HashToTxType(deliverTxBytes)

	s.Require().Equal(rosetta.DeliverTxTx, txType)
	s.Require().Equal(deliverTxBytes, hash, "deliver tx hash should not change")

	finalizeBlockTxBytes, err := hex.DecodeString(finalizeBlockTxHex)
	s.Require().NoError(err)

	txType, hash = s.c.ToSDK().HashToTxType(finalizeBlockTxBytes)
	s.Require().Equal(rosetta.FinalizeBlockTx, txType)
	s.Require().Equal(deliverTxBytes, hash, "end block tx hash should be equal to a block hash")

	txType, hash = s.c.ToSDK().HashToTxType([]byte("invalid"))

	s.Require().Equal(rosetta.UnrecognizedTx, txType)
	s.Require().Nil(hash)

	txType, hash = s.c.ToSDK().HashToTxType(append([]byte{0x3}, deliverTxBytes...))
	s.Require().Equal(rosetta.UnrecognizedTx, txType)
	s.Require().Nil(hash)
}

func (s *ConverterTestSuite) TestSigningComponents() {
	s.Run("invalid metadata coins", func() {
		_, _, err := s.c.ToRosetta().SigningComponents(nil, &rosetta.ConstructionMetadata{GasPrice: "invalid"}, nil)
		s.Require().ErrorIs(err, crgerrs.ErrConverter)
	})

	s.Run("length signers data does not match signers", func() {
		_, _, err := s.c.ToRosetta().SigningComponents(s.unsignedTx, &rosetta.ConstructionMetadata{GasPrice: "10stake"}, nil)
		s.Require().ErrorIs(err, crgerrs.ErrBadArgument)
	})

	s.Run("length pub keys does not match signers", func() {
		_, _, err := s.c.ToRosetta().SigningComponents(
			s.unsignedTx,
			&rosetta.ConstructionMetadata{GasPrice: "10stake", SignersData: []*rosetta.SignerData{
				{
					AccountNumber: 0,
					Sequence:      0,
				},
			}},
			nil)
		s.Require().ErrorIs(err, crgerrs.ErrBadArgument)
	})

	s.Run("ros pub key is valid but not the one we expect", func() {
		validButUnexpected, err := hex.DecodeString("030da9096a40eb1d6c25f1e26e9cbf8941fc84b8f4dc509c8df5e62a29ab8f2415")
		s.Require().NoError(err)

		_, _, err = s.c.ToRosetta().SigningComponents(
			s.unsignedTx,
			&rosetta.ConstructionMetadata{GasPrice: "10stake", SignersData: []*rosetta.SignerData{
				{
					AccountNumber: 0,
					Sequence:      0,
				},
			}},
			[]*rosettatypes.PublicKey{
				{
					Bytes:     validButUnexpected,
					CurveType: rosettatypes.Secp256k1,
				},
			})
		s.Require().ErrorIs(err, crgerrs.ErrBadArgument)
	})

	s.Run("success", func() {
		expectedPubKey, err := hex.DecodeString("034c92046950c876f4a5cb6c7797d6eeb9ef80d67ced4d45fb62b1e859240ba9ad")
		s.Require().NoError(err)

		_, _, err = s.c.ToRosetta().SigningComponents(
			s.unsignedTx,
			&rosetta.ConstructionMetadata{GasPrice: "10stake", SignersData: []*rosetta.SignerData{
				{
					AccountNumber: 0,
					Sequence:      0,
				},
			}},
			[]*rosettatypes.PublicKey{
				{
					Bytes:     expectedPubKey,
					CurveType: rosettatypes.Secp256k1,
				},
			})
		s.Require().NoError(err)
	})
}

func (s *ConverterTestSuite) TestBalanceOps() {
	s.Run("not a balance op", func() {
		notBalanceOp := abci.Event{
			Type: "not-a-balance-op",
		}

		ops := s.c.ToRosetta().BalanceOps("", []abci.Event{notBalanceOp})
		s.Len(ops, 0, "expected no balance ops")
	})

	// TODO - Investigate / fix sdk update discrepancies
	s.Run("multiple balance ops from 2 multicoins event", func() {
		subBalanceOp := bank.NewCoinSpentEvent(
			sdk.AccAddress("test"),
			sdk.NewCoins(sdk.NewInt64Coin("test", 10), sdk.NewInt64Coin("utxo", 10)),
		)

		addBalanceOp := bank.NewCoinReceivedEvent(
			sdk.AccAddress("test"),
			sdk.NewCoins(sdk.NewInt64Coin("test", 10), sdk.NewInt64Coin("utxo", 10)),
		)

		ops := s.c.ToRosetta().BalanceOps("", []abci.Event{(abci.Event)(subBalanceOp), (abci.Event)(addBalanceOp)})
		s.Len(ops, 4)
	})

	s.Run("spec broken", func() {
		s.Require().Panics(func() {
			specBrokenSub := abci.Event{
				Type: bank.EventTypeCoinSpent,
			}
			_ = s.c.ToRosetta().BalanceOps("", []abci.Event{specBrokenSub})
		})

		s.Require().Panics(func() {
			specBrokenSub := abci.Event{
				Type: bank.EventTypeCoinBurn,
			}
			_ = s.c.ToRosetta().BalanceOps("", []abci.Event{specBrokenSub})
		})

		s.Require().Panics(func() {
			specBrokenSub := abci.Event{
				Type: bank.EventTypeCoinReceived,
			}
			_ = s.c.ToRosetta().BalanceOps("", []abci.Event{specBrokenSub})
		})
	})
}

func (s *ConverterTestSuite) TestTxWithMemo() {
	addr1 := sdk.AccAddress("address1")
	addr2 := sdk.AccAddress("address2")

	msg := &bank.MsgSend{
		FromAddress: addr1.String(),
		ToAddress:   addr2.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
	}

	builder := s.txConf.NewTxBuilder()
	err := builder.SetMsgs(msg)
	s.Require().NoError(err)

	expectedMemo := "test-memo-for-deposit-123"
	builder.SetMemo(expectedMemo)

	txBytes, err := s.txConf.TxEncoder()(builder.GetTx())
	s.Require().NoError(err)

	rosTx, err := s.c.ToRosetta().Tx(txBytes, nil)
	s.Require().NoError(err)

	s.Require().NotNil(rosTx.Metadata)
	memo, ok := rosTx.Metadata["memo"]
	s.Require().True(ok, "metadata should contain memo field")
	s.Require().Equal(expectedMemo, memo)
}

func (s *ConverterTestSuite) TestTxWithEmptyMemo() {
	addr1 := sdk.AccAddress("address1")
	addr2 := sdk.AccAddress("address2")

	msg := &bank.MsgSend{
		FromAddress: addr1.String(),
		ToAddress:   addr2.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
	}

	builder := s.txConf.NewTxBuilder()
	err := builder.SetMsgs(msg)
	s.Require().NoError(err)

	txBytes, err := s.txConf.TxEncoder()(builder.GetTx())
	s.Require().NoError(err)

	rosTx, err := s.c.ToRosetta().Tx(txBytes, nil)
	s.Require().NoError(err)

	s.Require().NotNil(rosTx.Metadata)
	memo, ok := rosTx.Metadata["memo"]
	s.Require().True(ok, "metadata should contain memo field")
	s.Require().Equal("", memo)
}

func (s *ConverterTestSuite) TestTxMsgSendWithEvents() {
	// Build a MsgSend transaction
	sender := sdk.AccAddress("sender-address-bytes1")
	recipient := sdk.AccAddress("recip-address-bytes12")

	msg := &bank.MsgSend{
		FromAddress: sender.String(),
		ToAddress:   recipient.String(),
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("utest", 50000)),
	}

	builder := s.txConf.NewTxBuilder()
	s.Require().NoError(builder.SetMsgs(msg))

	txBytes, err := s.txConf.TxEncoder()(builder.GetTx())
	s.Require().NoError(err)

	feeCollectorAddr := rosetta.FeeCollector.String()

	s.Run("with tx result produces Transfer + balance ops + fee ops without fee duplicates", func() {
		txResult := &abci.ExecTxResult{
			Code: 0,
			Events: []abci.Event{
				// fee collection: coin_spent from sender (fee amount)
				{Type: bank.EventTypeCoinSpent, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeySpender, Value: sender.String()},
					{Key: sdk.AttributeKeyAmount, Value: "10utest"},
				}},
				// fee collection: coin_received by fee collector
				{Type: bank.EventTypeCoinReceived, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeyReceiver, Value: feeCollectorAddr},
					{Key: sdk.AttributeKeyAmount, Value: "10utest"},
				}},
				// fee event (tx-level)
				{Type: sdk.EventTypeTx, Attributes: []abci.EventAttribute{
					{Key: sdk.AttributeKeyFee, Value: "10utest"},
					{Key: sdk.AttributeKeyFeePayer, Value: sender.String()},
				}},
				// MsgSend: coin_spent from sender (transfer amount)
				{Type: bank.EventTypeCoinSpent, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeySpender, Value: sender.String()},
					{Key: sdk.AttributeKeyAmount, Value: "50000utest"},
				}},
				// MsgSend: coin_received by recipient (transfer amount)
				{Type: bank.EventTypeCoinReceived, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeyReceiver, Value: recipient.String()},
					{Key: sdk.AttributeKeyAmount, Value: "50000utest"},
				}},
			},
		}

		rosTx, err := s.c.ToRosetta().Tx(txBytes, txResult)
		s.Require().NoError(err)

		ops := rosTx.Operations

		// Collect op types for overview
		var types []string
		for _, op := range ops {
			types = append(types, op.Type)
		}

		// Should have: Transfer (from Ops) + coin_spent + coin_received (transfer balance ops) + fee_payer + fee_receiver
		// Should NOT have: coin_spent/coin_received for fee (those are deduplicated)
		s.Require().Equal(5, len(ops), "expected 5 ops (Transfer + 2 balance + 2 fee), got: %v", types)

		// First op: Transfer from Ops()
		s.Equal(rosetta.TransferOperation, ops[0].Type)
		s.Equal(sender.String(), ops[0].Account.Address)

		// Balance ops: coin_spent from sender (transfer amount)
		s.Equal(bank.EventTypeCoinSpent, ops[1].Type)
		s.Equal(sender.String(), ops[1].Account.Address)
		s.Equal("-50000", ops[1].Amount.Value)

		// Balance ops: coin_received by recipient (transfer amount)
		s.Equal(bank.EventTypeCoinReceived, ops[2].Type)
		s.Equal(recipient.String(), ops[2].Account.Address)
		s.Equal("50000", ops[2].Amount.Value)

		// Fee ops
		s.Equal(rosetta.FeePayerOperation, ops[3].Type)
		s.Equal(sender.String(), ops[3].Account.Address)
		s.Equal("-10", ops[3].Amount.Value)

		s.Equal(rosetta.FeeReceiverOperation, ops[4].Type)
		s.Equal(feeCollectorAddr, ops[4].Account.Address)
		s.Equal("10", ops[4].Amount.Value)

		// Verify no fee collector in coin_spent/coin_received ops
		for _, op := range ops {
			if op.Type == bank.EventTypeCoinSpent || op.Type == bank.EventTypeCoinReceived {
				s.NotEqual(feeCollectorAddr, op.Account.Address,
					"fee collector should not appear in coin_spent/coin_received when fee ops are present")
			}
		}
	})

	s.Run("nil tx result produces only Transfer op", func() {
		rosTx, err := s.c.ToRosetta().Tx(txBytes, nil)
		s.Require().NoError(err)

		ops := rosTx.Operations
		s.Require().Equal(1, len(ops))
		s.Equal(rosetta.TransferOperation, ops[0].Type)
		s.Equal(sender.String(), ops[0].Account.Address)
	})

	s.Run("failed tx produces Transfer op with reverted status", func() {
		txResult := &abci.ExecTxResult{
			Code: 1, // non-zero = failure
		}

		rosTx, err := s.c.ToRosetta().Tx(txBytes, txResult)
		s.Require().NoError(err)

		ops := rosTx.Operations
		// Failed tx: Ops() still runs but BalanceOps also runs (no events though)
		s.Require().Equal(1, len(ops))
		s.Equal(rosetta.TransferOperation, ops[0].Type)
		s.Require().NotNil(ops[0].Status)
		s.Equal(rosetta.StatusTxReverted, *ops[0].Status)
	})

	s.Run("no fee event means no fee ops and no deduplication", func() {
		txResult := &abci.ExecTxResult{
			Code: 0,
			Events: []abci.Event{
				// Only MsgSend events, no fee events
				{Type: bank.EventTypeCoinSpent, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeySpender, Value: sender.String()},
					{Key: sdk.AttributeKeyAmount, Value: "50000utest"},
				}},
				{Type: bank.EventTypeCoinReceived, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeyReceiver, Value: recipient.String()},
					{Key: sdk.AttributeKeyAmount, Value: "50000utest"},
				}},
			},
		}

		rosTx, err := s.c.ToRosetta().Tx(txBytes, txResult)
		s.Require().NoError(err)

		ops := rosTx.Operations
		// Transfer + coin_spent + coin_received (no fee ops, no dedup)
		s.Require().Equal(3, len(ops))
		s.Equal(rosetta.TransferOperation, ops[0].Type)
		s.Equal(bank.EventTypeCoinSpent, ops[1].Type)
		s.Equal(bank.EventTypeCoinReceived, ops[2].Type)
	})
}

func (s *ConverterTestSuite) TestTxMsgMultiSendSkipsOps() {
	// Build a MsgMultiSend transaction
	sender := sdk.AccAddress("sender-address-bytes1")
	recipient := sdk.AccAddress("recip-address-bytes12")

	msg := &bank.MsgMultiSend{
		Inputs: []bank.Input{
			{Address: sender.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("utest", 10000000000))},
		},
		Outputs: []bank.Output{
			{Address: recipient.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("utest", 10000000000))},
		},
	}

	builder := s.txConf.NewTxBuilder()
	s.Require().NoError(builder.SetMsgs(msg))

	txBytes, err := s.txConf.TxEncoder()(builder.GetTx())
	s.Require().NoError(err)

	feeCollectorAddr := rosetta.FeeCollector.String()

	s.Run("MsgMultiSend produces only balance ops and fee ops, no message-level op", func() {
		txResult := &abci.ExecTxResult{
			Code: 0,
			Events: []abci.Event{
				// fee collection
				{Type: bank.EventTypeCoinSpent, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeySpender, Value: sender.String()},
					{Key: sdk.AttributeKeyAmount, Value: "10utest"},
				}},
				{Type: bank.EventTypeCoinReceived, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeyReceiver, Value: feeCollectorAddr},
					{Key: sdk.AttributeKeyAmount, Value: "10utest"},
				}},
				{Type: sdk.EventTypeTx, Attributes: []abci.EventAttribute{
					{Key: sdk.AttributeKeyFee, Value: "10utest"},
					{Key: sdk.AttributeKeyFeePayer, Value: sender.String()},
				}},
				// MsgMultiSend balance events
				{Type: bank.EventTypeCoinSpent, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeySpender, Value: sender.String()},
					{Key: sdk.AttributeKeyAmount, Value: "10000000000utest"},
				}},
				{Type: bank.EventTypeCoinReceived, Attributes: []abci.EventAttribute{
					{Key: bank.AttributeKeyReceiver, Value: recipient.String()},
					{Key: sdk.AttributeKeyAmount, Value: "10000000000utest"},
				}},
			},
		}

		rosTx, err := s.c.ToRosetta().Tx(txBytes, txResult)
		s.Require().NoError(err)

		ops := rosTx.Operations

		var types []string
		for _, op := range ops {
			types = append(types, op.Type)
		}

		// Should have: coin_spent + coin_received (transfer) + fee_payer + fee_receiver
		// Should NOT have: /cosmos.bank.v1beta1.MsgMultiSend op
		s.Require().Equal(4, len(ops), "expected 4 ops (2 balance + 2 fee), got: %v", types)

		// Balance ops
		s.Equal(bank.EventTypeCoinSpent, ops[0].Type)
		s.Equal(sender.String(), ops[0].Account.Address)
		s.Equal("-10000000000", ops[0].Amount.Value)

		s.Equal(bank.EventTypeCoinReceived, ops[1].Type)
		s.Equal(recipient.String(), ops[1].Account.Address)
		s.Equal("10000000000", ops[1].Amount.Value)

		// Fee ops
		s.Equal(rosetta.FeePayerOperation, ops[2].Type)
		s.Equal(rosetta.FeeReceiverOperation, ops[3].Type)

		// No MsgMultiSend type op
		for _, op := range ops {
			s.NotEqual(rosetta.MsgMultiSendOperation, op.Type,
				"MsgMultiSend should not produce a message-level operation")
		}
	})
}

func TestConverterTestSuite(t *testing.T) {
	suite.Run(t, new(ConverterTestSuite))
}

// SymbolDecimalsTestSuite tests the symbol decimals mapping functionality
type SymbolDecimalsTestSuite struct {
	suite.Suite
}

func (s *SymbolDecimalsTestSuite) TestAmountsWithSymbolDecimals() {
	cdc, ir := rosetta.MakeCodec()
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	symbolDecimals := []rosetta.SymbolDecimal{
		{Base: "uaxl", Symbol: "AXL", Decimal: 6},
		{Base: "uatom", Symbol: "ATOM", Decimal: 8},
	}

	c := rosetta.NewConverter(cdc, ir, txConfig, address.NewBech32Codec("cosmos"), symbolDecimals)

	// ownedCoins: the actual balances the account holds
	ownedCoins := []sdk.Coin{
		sdk.NewInt64Coin("uatom", 2000000),
		sdk.NewInt64Coin("uaxl", 1000000),
		sdk.NewInt64Coin("unknown", 500),
		sdk.NewInt64Coin("ibc/ABC123", 999), // IBC token without mapping
	}

	amounts := c.ToRosetta().Amounts(ownedCoins)

	s.Require().Len(amounts, 4)

	// Results are ordered by ownedCoins order

	// uatom: owned with 2000000, mapped to ATOM
	s.Require().Equal("ATOM", amounts[0].Currency.Symbol)
	s.Require().Equal(int32(8), amounts[0].Currency.Decimals)
	s.Require().Equal("2000000", amounts[0].Value)

	// uaxl: owned with 1000000, mapped to AXL
	s.Require().Equal("AXL", amounts[1].Currency.Symbol)
	s.Require().Equal(int32(6), amounts[1].Currency.Decimals)
	s.Require().Equal("1000000", amounts[1].Value)

	// unknown: owned with 500, no mapping so stays as-is
	s.Require().Equal("unknown", amounts[2].Currency.Symbol)
	s.Require().Equal(int32(0), amounts[2].Currency.Decimals)
	s.Require().Equal("500", amounts[2].Value)

	// ibc/ABC123: owned with 999, no mapping so stays as-is
	s.Require().Equal("ibc/ABC123", amounts[3].Currency.Symbol)
	s.Require().Equal(int32(0), amounts[3].Currency.Decimals)
	s.Require().Equal("999", amounts[3].Value)
}

func (s *SymbolDecimalsTestSuite) TestBalanceOpsWithSymbolDecimals() {
	cdc, ir := rosetta.MakeCodec()
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	symbolDecimals := []rosetta.SymbolDecimal{
		{Base: "uaxl", Symbol: "AXL", Decimal: 6},
	}

	c := rosetta.NewConverter(cdc, ir, txConfig, address.NewBech32Codec("cosmos"), symbolDecimals)

	s.Run("coin_spent event with mapped denom", func() {
		spentEvent := bank.NewCoinSpentEvent(
			sdk.AccAddress("test"),
			sdk.NewCoins(sdk.NewInt64Coin("uaxl", 1000000)),
		)

		ops := c.ToRosetta().BalanceOps("success", []abci.Event{(abci.Event)(spentEvent)})

		s.Require().Len(ops, 1)
		s.Require().Equal("AXL", ops[0].Amount.Currency.Symbol)
		s.Require().Equal(int32(6), ops[0].Amount.Currency.Decimals)
		s.Require().Equal("-1000000", ops[0].Amount.Value)
	})

	s.Run("coin_received event with mapped denom", func() {
		receivedEvent := bank.NewCoinReceivedEvent(
			sdk.AccAddress("test"),
			sdk.NewCoins(sdk.NewInt64Coin("uaxl", 500000)),
		)

		ops := c.ToRosetta().BalanceOps("success", []abci.Event{(abci.Event)(receivedEvent)})

		s.Require().Len(ops, 1)
		s.Require().Equal("AXL", ops[0].Amount.Currency.Symbol)
		s.Require().Equal(int32(6), ops[0].Amount.Currency.Decimals)
		s.Require().Equal("500000", ops[0].Amount.Value)
	})

	s.Run("burn event with mapped denom", func() {
		burnEvent := bank.NewCoinBurnEvent(
			sdk.AccAddress("test"),
			sdk.NewCoins(sdk.NewInt64Coin("uaxl", 250000)),
		)

		ops := c.ToRosetta().BalanceOps("success", []abci.Event{(abci.Event)(burnEvent)})

		s.Require().Len(ops, 1)
		s.Require().Equal("AXL", ops[0].Amount.Currency.Symbol)
		s.Require().Equal(int32(6), ops[0].Amount.Currency.Decimals)
		s.Require().Equal("250000", ops[0].Amount.Value) // burn is not negated (sent to burner address)
	})

	s.Run("event with unmapped denom stays as-is", func() {
		spentEvent := bank.NewCoinSpentEvent(
			sdk.AccAddress("test"),
			sdk.NewCoins(sdk.NewInt64Coin("unknown", 100)),
		)

		ops := c.ToRosetta().BalanceOps("success", []abci.Event{(abci.Event)(spentEvent)})

		s.Require().Len(ops, 1)
		s.Require().Equal("unknown", ops[0].Amount.Currency.Symbol)
		s.Require().Equal(int32(0), ops[0].Amount.Currency.Decimals)
		s.Require().Equal("-100", ops[0].Amount.Value)
	})
}

func (s *SymbolDecimalsTestSuite) TestNoSymbolDecimals() {
	cdc, ir := rosetta.MakeCodec()
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	// Create converter without symbol decimals
	c := rosetta.NewConverter(cdc, ir, txConfig, address.NewBech32Codec("cosmos"), nil)

	// ownedCoins: the actual balances the account holds
	ownedCoins := []sdk.Coin{
		sdk.NewInt64Coin("uaxl", 1000000),
	}

	amounts := c.ToRosetta().Amounts(ownedCoins)

	s.Require().Len(amounts, 1)
	// Without symbol decimals, denom should remain unchanged
	s.Require().Equal("uaxl", amounts[0].Currency.Symbol)
	s.Require().Equal(int32(0), amounts[0].Currency.Decimals)
	s.Require().Equal("1000000", amounts[0].Value)
}

func (s *SymbolDecimalsTestSuite) TestAmountsWithMetadata() {
	cdc, ir := rosetta.MakeCodec()
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	symbolDecimals := []rosetta.SymbolDecimal{
		{Base: "uaxl", Symbol: "AXL", Decimal: 6},
	}

	c := rosetta.NewConverter(cdc, ir, txConfig, address.NewBech32Codec("axelar"), symbolDecimals)

	ownedCoins := []sdk.Coin{
		sdk.NewInt64Coin("uaxl", 1000000),
	}

	metadata := map[string]interface{}{
		"balance_type":      "delegated_balance",
		"validator_address": "axelarvaloper1abc",
	}

	amounts := c.ToRosetta().Amounts(ownedCoins, metadata)

	s.Require().Len(amounts, 1)
	s.Require().Equal("AXL", amounts[0].Currency.Symbol)
	s.Require().Equal(int32(6), amounts[0].Currency.Decimals)
	s.Require().Equal("1000000", amounts[0].Value)
	s.Require().NotNil(amounts[0].Metadata)
	s.Require().Equal("delegated_balance", amounts[0].Metadata["balance_type"])
	s.Require().Equal("axelarvaloper1abc", amounts[0].Metadata["validator_address"])
}

func TestSymbolDecimalsTestSuite(t *testing.T) {
	suite.Run(t, new(SymbolDecimalsTestSuite))
}
