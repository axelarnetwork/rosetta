# Axelar Customizations

This document details all customizations made to the [cosmos/rosetta](https://github.com/cosmos/rosetta) implementation for compatibility with Axelar network (axelar-core v1.3.x).

## Table of Contents

- [Overview](#overview)
- [New CLI Flag: `--symbol-decimals`](#new-cli-flag---symbol-decimals)
- [Account Balance Improvements](#account-balance-improvements)
- [Sub-Account Staking Queries](#sub-account-staking-queries)
- [Transaction Signing Fixes](#transaction-signing-fixes)
- [Symbol/Decimals Display Mapping](#symboldecimal-display-mapping)
- [Bug Fixes](#bug-fixes)
- [Upstream Compatibility](#upstream-compatibility)

---

## Overview

This branch (`axelar-core-v1.3.x-compatible`) contains customizations required to run the Rosetta API server against Axelar network nodes running axelar-core v1.3.x.

**Base branch:** `release/v0.50.x` (Cosmos SDK v0.50.12, compatible with axelar-core v1.3.x)

**Main customizations:**
1. Restoration of staking query features from cosmos-sdk v0.45 embedded rosetta
2. Symbol/decimals mapping for human-readable currency display
3. Various bug fixes for account balance and transaction construction

---

## New CLI Flag: `--symbol-decimals`

**What:** Maps base denominations to display symbols with decimal places

**Where:** `config.go:71`, `config.go:335-370`

**Usage:**
```shell
rosetta --symbol-decimals "uaxl:AXL:6,uusdc:USDC:6" ...
```

**Format:** Comma-separated entries of `base_denom:symbol:decimals`

**Why:** The Rosetta API returns currency information in responses. Without this mapping, currencies appear as raw base denoms (e.g., `uaxl`) with 0 decimals. This flag restores the human-readable display functionality that existed in the cosmos-sdk v0.45 embedded rosetta implementation.

**Effect:**
- Without flag: `{"symbol": "uaxl", "decimals": 0}`
- With flag: `{"symbol": "AXL", "decimals": 6}`

---

## Account Balance Improvements

### Return Only Owned Coins

**What changed:** The `/account/balance` endpoint now returns only coins the account actually owns, instead of all chain denominations with zero balances for unowned coins.

**Where:** `converter.go:433-448`, `client_online.go:212-221`

**Why:** The previous implementation called `TotalSupply` to get all available denoms, then returned zero balances for coins the account didn't own. This was:
1. Inefficient (required extra RPC calls)
2. Confusing (returned many zero-balance entries)
3. Buggy (the pagination in `coins()` function was incorrect)

### Sequence Number in Metadata

**What changed:** Account balance responses now include `sequence_number` in metadata.

**Where:** `lib/internal/service/account.go:81-92`

**Response format:**
```json
{
  "block_identifier": {...},
  "balances": [...],
  "metadata": {
    "sequence_number": 42
  }
}
```

**Why:** The sequence number is essential for transaction construction. Including it in balance responses reduces the number of RPC calls clients need to make.

### Height Bug Fix

**What changed:** Fixed issue where balance queries at a specific height were actually querying height 0.

**Where:** `lib/internal/service/account.go:29-45`

**Why:** The height variable was being set from block index after the balance query instead of before, causing incorrect historical balance queries.

---

## Sub-Account Staking Queries

**What:** Added support for querying staking-related balances via the standard `/account/balance` endpoint using sub-account identifiers.

**Where:**
- `lib/types/types.go:10-72` - BalanceType enum and helpers
- `lib/internal/service/account.go:46-80` - Sub-account routing logic
- `lib/internal/service/types.go` - SubAccountMetaData struct
- `client_online.go:199-267` - New client methods

**Why:** This restores functionality from the cosmos-sdk v0.45 embedded rosetta that was lost in the standalone rosetta implementation.

### Available Sub-Account Types

#### `delegated_balance`
Query delegated tokens across all validators:
```json
{
  "account_identifier": {
    "address": "axelar1...",
    "sub_account": {
      "address": "delegated_balance"
    }
  }
}
```

Response includes metadata with validator address for each delegation.

#### `unbonding_balance`
Query tokens currently unbonding:
```json
{
  "account_identifier": {
    "address": "axelar1...",
    "sub_account": {
      "address": "unbonding_balance"
    }
  }
}
```

Response includes metadata with validator address and completion time for each unbonding entry.

#### `pending_rewards`
Query unclaimed staking rewards:
```json
{
  "account_identifier": {
    "address": "axelar1...",
    "sub_account": {
      "address": "pending_rewards"
    }
  }
}
```

Optionally filter by validator:
```json
{
  "account_identifier": {
    "address": "axelar1...",
    "sub_account": {
      "address": "pending_rewards",
      "metadata": {
        "validator_address": "axelarvaloper1..."
      }
    }
  }
}
```

### New Client Interface Methods

Added to `lib/types/types.go:108-119`:
- `AccountSequence(ctx, addr, height)` - Get account sequence number
- `ToCurrency(denom)` - Convert denom to Rosetta Currency with symbol mapping
- `Delegations(ctx, delegator, height)` - Get delegations
- `UnbondingDelegations(ctx, delegator, height)` - Get unbonding delegations
- `Rewards(ctx, delegator, validator, height)` - Get pending rewards

---

## Transaction Signing Fixes

### Sign Mode Change

**What changed:** Changed signing mode from `SIGN_MODE_DIRECT` to `SIGN_MODE_LEGACY_AMINO_JSON`

**Where:** `converter.go:134`

**Why:** `SIGN_MODE_DIRECT` uses protobuf encoding which can cause issues with certain clients and hardware wallets. Amino JSON is more widely supported and provides better compatibility.

### Signer Address Encoding

**What changed:** Signer addresses are now properly bech32-encoded instead of being returned as raw bytes.

**Where:** `converter.go:604-611`

**Why:** The previous implementation returned raw byte addresses which were not human-readable and couldn't be matched with account identifiers in Rosetta operations.

### Message Signer Method

**What changed:** Changed from `GetMsgSigners` to `GetMsgV1Signers`

**Where:** `converter.go:174`, `converter.go:254`

**Why:** `GetMsgV1Signers` is the correct method for cosmos-sdk v0.50.x message signer extraction.

---

## Symbol/Decimal Display Mapping

**What:** All Rosetta endpoints that return currency information now apply the symbol/decimals mapping configured via `--symbol-decimals`.

**Affected endpoints:**
- `/account/balance` - Balance amounts
- `/block` - Transaction operation amounts
- `/construction/metadata` - Suggested fees

**Where:**
- `converter.go:344-357` - `ToCurrency()` method
- `converter.go:423` - Balance operations
- `lib/internal/service/construction.go:104-108` - Suggested fees

---

## Bug Fixes

### Default Height Handling

**What:** When height is 0, it now defaults to latest block (nil) instead of explicitly querying height 0.

**Where:** `client_online.go:332-335`, `client_online.go:567-570`

**Why:** Height 0 is typically used to mean "latest" in API requests, but was being interpreted literally, causing "block not found" errors.

### Base64 Decoding Removal

**What:** Removed unnecessary base64 decoding for coin burn event attributes.

**Where:** `converter.go:397-399`

**Why:** Event attributes in cosmos-sdk v0.50.x are not base64 encoded, so decoding them was causing parse errors.

### TendermintRPC Websocket Path

**What:** Added explicit websocket path for TendermintRPC client initialization.

**Where:** `client_online.go:40`, `client_online.go:110`

**Why:** Required for proper RPC client initialization in newer CometBFT versions.

### Transaction Message Parsing Fix

**What:** Fixed `parseTxMessages` to properly marshal transaction messages instead of incorrectly using public keys.

**Where:** `utils.go:76-91`

**Why:** The original implementation was using `tx.GetPubKeys()` instead of `tx.GetMsgs()`, which returned incorrect data for transaction construction.

### Fee and Timeout Fields

**What:**
- Removed fee payer field (defaults to signer, cannot get exact value)
- Changed from `TimeoutTimestamp` to `TimeoutHeight` for SDK v0.50.x compatibility

**Where:** `utils.go:115`, `utils.go:131`

**Why:** API differences between cosmos-sdk v0.52.x and v0.50.x.

---

## File Change Summary

| File | Changes |
|------|---------|
| `config.go` | Added `--symbol-decimals` flag and parsing |
| `config_test.go` | Tests for symbol-decimals parsing (new file) |
| `converter.go` | Sign mode fix, signer encoding, ToCurrency, Amounts refactor, base64 removal |
| `converter_test.go` | Extended tests for converter changes |
| `client_online.go` | Staking queries, balance fixes, height handling, websocket path |
| `lib/internal/service/account.go` | Sub-account routing, sequence number |
| `lib/internal/service/construction.go` | Symbol-decimals for suggested fees |
| `lib/internal/service/types.go` | SubAccountMetaData struct (new file) |
| `lib/internal/service/types_test.go` | Tests for SubAccountMetaData (new file) |
| `lib/types/types.go` | BalanceType enum, new Client interface methods |
| `lib/types/types_test.go` | Tests for BalanceType (new file) |
| `utils.go` | Message parsing fix, fee/timeout field changes |

---

## Upstream Compatibility

This branch is based on `release/v0.50.x` (Cosmos SDK v0.50.12) and diverges from `main` (Cosmos SDK v0.52.x).

**When syncing with upstream:**
- Cherry-pick relevant fixes from `main` to this branch
- Be aware of API differences between SDK v0.50.x and v0.52.x
- Test thoroughly after any upstream sync

**To see Axelar-specific changes:**
```shell
git diff origin/release/v0.50.x..HEAD
```
