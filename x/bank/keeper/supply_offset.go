package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

// SupplyOffset is needed when we mint/burn tokens that are not part of the main supply.
// When minting and burning tokens that are not part of the main supply we offset it using
// these functions.

// The following code has been adapted from osmosis/cosmos-sdk fork
// https://github.com/osmosis-labs/cosmos-sdk/blob/2b41ee894c5d5d61355aa0f366c0e164d3ba3451/x/bank/keeper/supply_offset.go

// GetSupplyOffset retrieves the SupplyOffset from store for a specific denom.
func (k BaseViewKeeper) GetSupplyOffset(ctx context.Context, denom string) math.Int {
	amt, err := k.SupplyOffset.Get(ctx, denom)
	if err != nil {
		return math.ZeroInt()
	}
	return amt
}

// setSupplyOffset sets the supply offset for the given denom.
func (k BaseKeeper) setSupplyOffset(ctx context.Context, denom string, offsetAmount math.Int) {
	_ = k.SupplyOffset.Set(ctx, denom, offsetAmount)

	// Bank invariants and IBC requires to remove zero coins.
	if offsetAmount.IsZero() {
		_ = k.SupplyOffset.Remove(ctx, denom)
	} else {
		_ = k.SupplyOffset.Set(ctx, denom, offsetAmount)
	}
}

// AddSupplyOffset adjusts the current supply offset of a denom by the inputted offsetAmount.
func (k BaseKeeper) AddSupplyOffset(ctx context.Context, denom string, offsetAmount math.Int) {
	k.setSupplyOffset(ctx, denom, k.GetSupplyOffset(ctx, denom).Add(offsetAmount))
}

// GetSupplyWithOffset retrieves the Supply of a denom and offsets it by SupplyOffset
// If SupplyWithOffset is negative, it returns 0. This is because sdk.Coin is not valid
// with a negative amount.
func (k BaseKeeper) GetSupplyWithOffset(ctx context.Context, denom string) sdk.Coin {
	supply := k.GetSupply(ctx, denom)
	supply.Amount = supply.Amount.Add(k.GetSupplyOffset(ctx, denom))

	if supply.Amount.IsNegative() {
		supply.Amount = math.ZeroInt()
	}

	return supply
}

// GetPaginatedTotalSupplyWithOffsets queries for the supply with offset, ignoring 0 coins, with a given pagination.
func (k BaseKeeper) GetPaginatedTotalSupplyWithOffsets(
	ctx context.Context,
	pagination *query.PageRequest,
) (sdk.Coins, *query.PageResponse, error) {
	coins, pageResp, err := query.CollectionPaginate(ctx, k.Supply, pagination, func(key string, value math.Int) (sdk.Coin, error) {

		// `Add` omits the 0 coins addition to the `supply`.
		value = value.Add(k.GetSupplyOffset(ctx, key))

		return sdk.NewCoin(key, value), nil
	})
	if err != nil {
		return nil, nil, err
	}

	return coins, pageResp, nil
}

// IterateTotalSupplyWithOffsets iterates over the total supply with offsets calling the given cb (callback) function
// with the balance of each coin. The iteration stops if the callback returns true.
func (k BaseKeeper) IterateTotalSupplyWithOffsets(ctx context.Context, cb func(sdk.Coin) bool) {
	err := k.Supply.Walk(ctx, nil, func(s string, m math.Int) (bool, error) {
		return cb(sdk.NewCoin(s, m.Add(k.GetSupplyOffset(ctx, s)))), nil
	})
	if err != nil {
		panic(err)
	}
}
