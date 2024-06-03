package keeper_test

import (
	"cosmossdk.io/math"
)

func (suite *KeeperTestSuite) TestGetSupplyOffset() {
	bk := suite.bankKeeper
	offset := bk.GetSupplyOffset(suite.ctx, "denom")
	suite.Require().True(offset.IsZero())

	// Set an offset and retrieve it
	bk.SetSupplyOffset(suite.ctx, "denom", math.NewInt(100))
	offset = bk.GetSupplyOffset(suite.ctx, "denom")

	suite.Require().Equal(math.NewInt(100), offset)
}

//func TestSetSupplyOffset(t *testing.T) {
//	ctx, keeper := setupTest(t)
//
//	// Set an offset
//	keeper.setSupplyOffset(ctx, "denom", math.NewInt(100))
//
//	// Verify it is stored correctly
//	store := ctx.KVStore(keeper.storeKey)
//	supplyOffsetStore := prefix.NewStore(store, types.SupplyOffsetKey)
//	bz := supplyOffsetStore.Get([]byte("denom"))
//	var amount math.Int
//	err := amount.Unmarshal(bz)
//	require.NoError(t, err)
//	require.Equal(t, math.NewInt(100), amount)
//
//	// Set offset to zero and verify it is deleted
//	keeper.setSupplyOffset(ctx, "denom", math.ZeroInt())
//	bz = supplyOffsetStore.Get([]byte("denom"))
//	require.Nil(t, bz)
//}
//
//func TestAddSupplyOffset(t *testing.T) {
//	ctx, keeper := setupTest(t)
//
//	// Add an offset
//	keeper.AddSupplyOffset(ctx, "denom", math.NewInt(100))
//	offset := keeper.GetSupplyOffset(ctx, "denom")
//	require.Equal(t, math.NewInt(100), offset)
//
//	// Add more to the offset
//	keeper.AddSupplyOffset(ctx, "denom", math.NewInt(50))
//	offset = keeper.GetSupplyOffset(ctx, "denom")
//	require.Equal(t, math.NewInt(150), offset)
//}
//
//func TestGetSupplyWithOffset(t *testing.T) {
//	ctx, keeper := setupTest(t)
//
//	// Assume GetSupply is implemented and returns 100 for "denom"
//	// Set the mock return value of GetSupply
//	keeper.setSupplyOffset(ctx, "denom", math.NewInt(50))
//
//	// Retrieve supply with offset
//	supplyWithOffset := keeper.GetSupplyWithOffset(ctx, "denom")
//	expectedSupply := sdk.NewCoin("denom", math.NewInt(150))
//	require.Equal(t, expectedSupply, supplyWithOffset)
//
//	// Test negative offset resulting in zero supply
//	keeper.setSupplyOffset(ctx, "denom", math.NewInt(-200))
//	supplyWithOffset = keeper.GetSupplyWithOffset(ctx, "denom")
//	expectedSupply = sdk.NewCoin("denom", math.ZeroInt())
//	require.Equal(t, expectedSupply, supplyWithOffset)
//}
//
//func TestGetPaginatedTotalSupplyWithOffsets(t *testing.T) {
//	ctx, keeper := setupTest(t)
//
//	// Assume there are multiple denoms with supplies
//	// Add supplies and offsets
//	keeper.setSupplyOffset(ctx, "denom1", math.NewInt(50))
//	keeper.setSupplyOffset(ctx, "denom2", math.NewInt(-30))
//
//	// Retrieve paginated supply with offsets
//	pageRequest := &query.PageRequest{Limit: 10, Offset: 0}
//	totalSupply, _, err := keeper.GetPaginatedTotalSupplyWithOffsets(ctx, pageRequest)
//	require.NoError(t, err)
//
//	// Verify the results
//	expectedSupply := sdk.NewCoins(
//		sdk.NewCoin("denom1", math.NewInt(150)), // Assuming GetSupply returns 100
//		sdk.NewCoin("denom2", math.NewInt(70)),  // Assuming GetSupply returns 100
//	)
//	require.Equal(t, expectedSupply, totalSupply)
//}
//
//func TestIterateTotalSupplyWithOffsets(t *testing.T) {
//	ctx, keeper := setupTest(t)
//
//	// Add supplies and offsets
//	keeper.setSupplyOffset(ctx, "denom1", math.NewInt(50))
//	keeper.setSupplyOffset(ctx, "denom2", math.NewInt(-30))
//
//	// Iterate over the total supply with offsets
//	expectedSupplies := map[string]math.Int{
//		"denom1": math.NewInt(150), // Assuming GetSupply returns 100
//		"denom2": math.NewInt(70),  // Assuming GetSupply returns 100
//	}
//	keeper.IterateTotalSupplyWithOffsets(ctx, func(coin sdk.Coin) bool {
//		expectedAmount, ok := expectedSupplies[coin.Denom]
//		require.True(t, ok)
//		require.Equal(t, expectedAmount, coin.Amount)
//		delete(expectedSupplies, coin.Denom)
//		return false
//	})
//
//	// Verify all expected supplies were iterated
//	require.Empty(t, expectedSupplies)
//}
