package keeper_test

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/runtime"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
)

var _ types.BankHooks = &MockBankHooksReceiver{}

type testingSuite struct {
	BankKeeper    bankkeeper.Keeper
	AccountKeeper types.AccountKeeper
	StakingKeeper stakingkeeper.Keeper
	App           *runtime.App
}

// BankHooks event hooks for bank (noalias)
type MockBankHooksReceiver struct{}

// Mock BlockBeforeSend bank hook that doesn't allow the sending of exactly 100 coins of any denom.
func (h *MockBankHooksReceiver) BlockBeforeSend(ctx context.Context, from, to sdk.AccAddress, amount sdk.Coins) error {
	for _, coin := range amount {
		if coin.Amount.Equal(math.NewInt(100)) {
			return fmt.Errorf("not allowed; expected %v, got: %v", 100, coin.Amount)
		}
	}
	return nil
}

// variable for counting `TrackBeforeSend`
var (
	countTrackBeforeSend = 0
	expNextCount         = 1
)

// Mock TrackBeforeSend bank hook that simply tracks the sending of exactly 50 coins of any denom.
func (h *MockBankHooksReceiver) TrackBeforeSend(ctx context.Context, from, to sdk.AccAddress, amount sdk.Coins) {
	for _, coin := range amount {
		if coin.Amount.Equal(math.NewInt(50)) {
			countTrackBeforeSend += 1
		}
	}
}

func (suite *KeeperTestSuite) TestHooks() {
	ctx := suite.ctx
	require := suite.Require()
	balances := sdk.NewCoins(newFooCoin(1000))
	bk := suite.bankKeeper

	// Create a valid send amount which is 1 coin, and an invalidSendAmount which is 100 coins
	validSendAmount := sdk.NewCoins(newFooCoin(1))
	triggerTrackSendAmount := sdk.NewCoins(newFooCoin(50))
	invalidBlockSendAmount := sdk.NewCoins(newFooCoin(100))

	// setup our mock bank hooks receiver that prevents the send of 100 coins
	bankHooksReceiver := MockBankHooksReceiver{}
	bk.SetHooks(
		types.NewMultiBankHooks(&bankHooksReceiver),
	)

	suite.mockFundAccount(accAddrs[0])
	require.NoError(banktestutil.FundAccount(ctx, suite.bankKeeper, accAddrs[0], balances))

	acc0 := authtypes.NewBaseAccountWithAddress(accAddrs[0])
	acc1Balances := suite.bankKeeper.GetAllBalances(ctx, accAddrs[0])
	require.Equal(balances, acc1Balances)

	// try sending a validSendAmount and it should work
	suite.mockSendCoins(ctx, acc0, accAddrs[1])
	err := bk.SendCoins(ctx, accAddrs[0], accAddrs[1], validSendAmount)
	require.NoError(err)

	// try sending an trigger track send amount and it should work
	suite.mockSendCoins(ctx, acc0, accAddrs[1])
	err = bk.SendCoins(ctx, accAddrs[0], accAddrs[1], triggerTrackSendAmount)
	require.NoError(err)

	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++

	// try sending an invalidSendAmount and it should not work
	err = bk.SendCoins(ctx, accAddrs[0], accAddrs[1], invalidBlockSendAmount)
	require.Error(err)

	// try doing SendManyCoins and make sure if even a single sub-send is invalid, the entire function fails
	err = bk.SendManyCoins(ctx, accAddrs[0], []sdk.AccAddress{accAddrs[0], accAddrs[1]}, []sdk.Coins{invalidBlockSendAmount, validSendAmount})
	require.Error(err)

	suite.mockSendManyCoins(ctx, acc0)
	err = bk.SendManyCoins(ctx, accAddrs[0], []sdk.AccAddress{accAddrs[0], accAddrs[1]}, []sdk.Coins{triggerTrackSendAmount, validSendAmount})
	require.NoError(err)
	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++

	// make sure that account to module doesn't bypass hook
	suite.mockSendCoinsFromAccountToModule(acc0, burnerAcc)
	err = bk.SendCoinsFromAccountToModule(ctx, accAddrs[0], authtypes.Burner, validSendAmount)
	require.NoError(err)

	suite.authKeeper.EXPECT().GetModuleAccount(ctx, burnerAcc.Name).Return(burnerAcc)
	err = bk.SendCoinsFromAccountToModule(ctx, accAddrs[0], authtypes.Burner, invalidBlockSendAmount)
	require.Error(err)

	suite.mockSendCoinsFromAccountToModule(acc0, burnerAcc)
	err = bk.SendCoinsFromAccountToModule(ctx, accAddrs[0], authtypes.Burner, triggerTrackSendAmount)
	require.NoError(err)
	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++

	// Mint tokens to the module
	suite.mockMintCoins(minterAcc)
	require.NoError(bk.MintCoins(ctx, authtypes.Minter, balances))

	// make sure that module to account doesn't bypass hook
	suite.mockSendCoinsFromModuleToAccount(minterAcc, accAddrs[0])
	err = bk.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, accAddrs[0], validSendAmount)
	require.NoError(err)

	suite.authKeeper.EXPECT().GetModuleAddress(minterAcc.Name).Return(minterAcc.GetAddress())
	err = bk.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, accAddrs[0], invalidBlockSendAmount)
	require.Error(err)

	suite.mockSendCoinsFromModuleToAccount(minterAcc, accAddrs[0])
	err = bk.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, accAddrs[0], triggerTrackSendAmount)
	require.NoError(err)

	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++

	// make sure that module to module doesn't bypass hook
	suite.mockSendCoinsFromModuleToModule(minterAcc, holderAcc)
	err = bk.SendCoinsFromModuleToModule(ctx, authtypes.Minter, holderAcc.GetName(), validSendAmount)
	require.NoError(err)

	suite.mockSendCoinsFromModuleToModule(minterAcc, holderAcc)
	err = bk.SendCoinsFromModuleToModule(ctx, authtypes.Minter, holderAcc.GetName(), invalidBlockSendAmount)
	// there should be no error since module to module does not call block before send hooks
	require.NoError(err)

	suite.mockSendCoinsFromModuleToModule(minterAcc, holderAcc)
	err = bk.SendCoinsFromModuleToModule(ctx, authtypes.Minter, holderAcc.GetName(), triggerTrackSendAmount)
	require.NoError(err)

	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++

	// make sure that module to many accounts doesn't bypass hook
	suite.mockSendCoinsFromModuleToManyAccounts(minterAcc)
	err = bk.SendCoinsFromModuleToManyAccounts(ctx, authtypes.Minter, []sdk.AccAddress{accAddrs[0], accAddrs[1]}, []sdk.Coins{validSendAmount, validSendAmount})
	require.NoError(err)

	suite.authKeeper.EXPECT().GetModuleAddress(minterAcc.Name).Return(minterAcc.GetAddress())
	err = bk.SendCoinsFromModuleToManyAccounts(ctx, authtypes.Minter, []sdk.AccAddress{accAddrs[0], accAddrs[1]}, []sdk.Coins{validSendAmount, invalidBlockSendAmount})
	require.Error(err)

	suite.mockSendCoinsFromModuleToManyAccounts(minterAcc)
	err = bk.SendCoinsFromModuleToManyAccounts(ctx, authtypes.Minter, []sdk.AccAddress{accAddrs[0], accAddrs[1]}, []sdk.Coins{validSendAmount, triggerTrackSendAmount})
	require.NoError(err)

	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++

	// make sure that DelegateCoins doesn't bypass the hook
	suite.mockDelegateCoins(ctx, acc0, minterAcc)
	err = bk.DelegateCoins(ctx, accAddrs[0], minterAcc.GetAddress(), validSendAmount)
	require.NoError(err)

	suite.authKeeper.EXPECT().GetAccount(ctx, minterAcc.GetAddress()).Return(minterAcc)
	err = bk.DelegateCoins(ctx, accAddrs[0], minterAcc.GetAddress(), invalidBlockSendAmount)
	require.Error(err)

	suite.mockDelegateCoins(ctx, acc0, minterAcc)
	err = bk.DelegateCoins(ctx, accAddrs[0], minterAcc.GetAddress(), triggerTrackSendAmount)
	require.NoError(err)

	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++

	// make sure that UndelegateCoins doesn't bypass the hook
	suite.mockUnDelegateCoins(ctx, acc0, minterAcc)
	err = bk.UndelegateCoins(ctx, minterAcc.GetAddress(), accAddrs[0], validSendAmount)
	require.NoError(err)

	suite.authKeeper.EXPECT().GetAccount(ctx, minterAcc.GetAddress()).Return(minterAcc)
	err = bk.UndelegateCoins(ctx, minterAcc.GetAddress(), accAddrs[0], invalidBlockSendAmount)
	require.Error(err)

	suite.mockUnDelegateCoins(ctx, acc0, minterAcc)
	err = bk.UndelegateCoins(ctx, minterAcc.GetAddress(), accAddrs[0], triggerTrackSendAmount)
	require.NoError(err)

	require.Equal(countTrackBeforeSend, expNextCount)
	expNextCount++
}
