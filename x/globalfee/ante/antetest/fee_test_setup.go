package antetest

import (
	"fmt"

	tmrand "github.com/cometbft/cometbft/libs/rand"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"

	"github.com/White-Whale-Defi-Platform/migaloo-chain/v3/app/helpers"
	appparams "github.com/White-Whale-Defi-Platform/migaloo-chain/v3/app/params"
	feeante "github.com/White-Whale-Defi-Platform/migaloo-chain/v3/x/globalfee/ante"

	"github.com/White-Whale-Defi-Platform/migaloo-chain/v3/app"
	"github.com/White-Whale-Defi-Platform/migaloo-chain/v3/x/globalfee"
	globalfeetypes "github.com/White-Whale-Defi-Platform/migaloo-chain/v3/x/globalfee/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	app       *app.MigalooApp
	ctx       sdk.Context
	clientCtx client.Context
	txBuilder client.TxBuilder
}

var (
	testBondDenom                              = "uatom"
	testMaxTotalBypassMinFeeMsgGasUsage uint64 = 1_000_000
)

func (s *IntegrationTestSuite) SetupTest() {
	app := helpers.Setup(s.T())
	ctx := app.BaseApp.NewContext(false, tmproto.Header{
		ChainID: fmt.Sprintf("test-chain-%s", tmrand.Str(4)),
		Height:  1,
	})

	encodingConfig := appparams.MakeEncodingConfig()
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	testdata.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	s.app = app
	s.ctx = ctx
	s.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
}

func (s *IntegrationTestSuite) SetupTestGlobalFeeStoreAndMinGasPrice(minGasPrice []sdk.DecCoin, globalFeeParams *globalfeetypes.Params) (feeante.FeeDecorator, sdk.AnteHandler) {
	subspace := s.app.GetSubspace(globalfee.ModuleName)
	subspace.SetParamSet(s.ctx, globalFeeParams)
	s.ctx = s.ctx.WithMinGasPrices(minGasPrice).WithIsCheckTx(true)

	// set staking params
	stakingParam := stakingtypes.DefaultParams()
	stakingParam.BondDenom = testBondDenom
	stakingSubspace := s.SetupTestStakingSubspace(stakingParam)

	// build fee decorator
	feeDecorator := feeante.NewFeeDecorator(app.GetDefaultBypassFeeMessages(), subspace, stakingSubspace, uint64(1_000_000))

	// chain fee decorator to antehandler
	antehandler := sdk.ChainAnteDecorators(feeDecorator)

	return feeDecorator, antehandler
}

// SetupTestStakingSubspace sets uatom as bond denom for the fee tests.
func (s *IntegrationTestSuite) SetupTestStakingSubspace(params stakingtypes.Params) types.Subspace {
	s.app.GetSubspace(stakingtypes.ModuleName).SetParamSet(s.ctx, &params)
	return s.app.GetSubspace(stakingtypes.ModuleName)
}

func (s *IntegrationTestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (xauthsigning.Tx, error) {
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  s.clientCtx.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}

	if err := s.txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}

	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
		}
		sigV2, err := tx.SignWithPrivKey(
			s.clientCtx.TxConfig.SignModeHandler().DefaultMode(),
			signerData,
			s.txBuilder,
			priv,
			s.clientCtx.TxConfig,
			accSeqs[i],
		)
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}

	if err := s.txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}

	return s.txBuilder.GetTx(), nil
}
