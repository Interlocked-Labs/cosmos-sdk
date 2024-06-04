package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (suite *KeeperTestSuite) TestGetSetSupplyOffset() {
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper

	offset := bk.GetSupplyOffset(ctx, "denom")
	require.True(offset.IsZero())

	// Set an offset and retrieve it
	bk.SetSupplyOffset(ctx, "denom", math.NewInt(100))
	offset = bk.GetSupplyOffset(ctx, "denom")

	require.Equal(math.NewInt(100), offset)
}

func (suite *KeeperTestSuite) GetSupplyWithOffset() {
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper

	initCoins := sdk.NewCoins(
		sdk.NewCoin("denom1", math.NewInt(100)),
		sdk.NewCoin("denom2", math.NewInt(100)),
	)

	suite.mockMintCoins(minterAcc)
	require.NoError(bk.MintCoins(ctx, authtypes.Minter, initCoins))

	bk.SetSupplyOffset(suite.ctx, "denom1", math.NewInt(50))
	bk.SetSupplyOffset(suite.ctx, "denom2", math.NewInt(-30))

	// Set an offset and retrieve it
	denom1 := bk.GetSupplyWithOffset(ctx, "denom1")
	denom2 := bk.GetSupplyWithOffset(ctx, "denom2")

	require.Equal(denom1.Amount, math.NewInt(150))
	require.Equal(denom2.Amount, math.NewInt(70))
}

func (suite *KeeperTestSuite) TestAddSupplyOffset() {
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper

	// Add an offset
	bk.AddSupplyOffset(ctx, "denom", math.NewInt(100))
	offset := bk.GetSupplyOffset(ctx, "denom")
	require.Equal(math.NewInt(100), offset)

	// Add more to the offset
	bk.AddSupplyOffset(ctx, "denom", math.NewInt(50))
	offset = bk.GetSupplyOffset(ctx, "denom")
	require.Equal(math.NewInt(150), offset)
}

func (suite *KeeperTestSuite) TestGetPaginatedTotalSupplyWithOffsets() {
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper

	initCoins := sdk.NewCoins(
		sdk.NewCoin("denom1", math.NewInt(100)),
		sdk.NewCoin("denom2", math.NewInt(100)),
	)

	suite.mockMintCoins(minterAcc)
	require.NoError(bk.MintCoins(ctx, authtypes.Minter, initCoins))

	bk.SetSupplyOffset(suite.ctx, "denom1", math.NewInt(50))
	bk.SetSupplyOffset(suite.ctx, "denom2", math.NewInt(-30))

	// Retrieve paginated supply with offsets
	pageRequest := &query.PageRequest{Limit: 10, Offset: 0}
	totalSupply, _, err := bk.GetPaginatedTotalSupplyWithOffsets(suite.ctx, pageRequest)
	require.NoError(err)

	// Verify the results
	expectedSupply := sdk.NewCoins(
		sdk.NewCoin("denom1", math.NewInt(150)),
		sdk.NewCoin("denom2", math.NewInt(70)),
	)
	require.Equal(expectedSupply, totalSupply)
}

func (suite *KeeperTestSuite) TestIterateTotalSupplyWithOffsets() {
	ctx := suite.ctx
	require := suite.Require()
	bk := suite.bankKeeper

	initCoins := sdk.NewCoins(
		sdk.NewCoin("denom1", math.NewInt(100)),
		sdk.NewCoin("denom2", math.NewInt(100)),
	)

	suite.mockMintCoins(minterAcc)
	require.NoError(bk.MintCoins(ctx, authtypes.Minter, initCoins))

	// Add supplies and offsets
	bk.SetSupplyOffset(suite.ctx, "denom1", math.NewInt(50))
	bk.SetSupplyOffset(suite.ctx, "denom2", math.NewInt(-30))

	// Iterate over the total supply with offsets
	expectedSupplies := map[string]math.Int{
		"denom1": math.NewInt(150),
		"denom2": math.NewInt(70),
	}
	bk.IterateTotalSupplyWithOffsets(suite.ctx, func(coin sdk.Coin) bool {
		expectedAmount, ok := expectedSupplies[coin.Denom]
		require.True(ok)
		require.Equal(expectedAmount, coin.Amount)
		delete(expectedSupplies, coin.Denom)
		return false
	})

	// Verify all expected supplies were iterated
	require.Empty(expectedSupplies)
}
