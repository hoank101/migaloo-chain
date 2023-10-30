package ante

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	migaloofeeante "github.com/White-Whale-Defi-Platform/migaloo-chain/v3/x/globalfee/ante"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	Codec                  codec.BinaryCodec
	GovKeeper              *govkeeper.Keeper
	IBCkeeper              *ibckeeper.Keeper
	ExtensionOptionChecker authante.ExtensionOptionChecker
	BypassMinFeeMsgTypes   []string
	GlobalFeeSubspace      paramtypes.Subspace
	StakingSubspace        paramtypes.Subspace
	TxFeeChecker           authante.TxFeeChecker
}

func NewAnteHandler(opts HandlerOptions) (sdk.AnteHandler, error) {
	if opts.AccountKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if opts.BankKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if opts.SignModeHandler == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if opts.IBCkeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "IBC keeper is required for AnteHandler")
	}
	if opts.GlobalFeeSubspace.Name() == "" {
		return nil, sdkerrors.Wrap(sdkerrors.ErrNotFound, "globalfee param store is required for AnteHandler")
	}
	if opts.StakingSubspace.Name() == "" {
		return nil, sdkerrors.Wrap(sdkerrors.ErrNotFound, "staking param store is required for AnteHandler")
	}
	if opts.GovKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "gov keeper is required for AnteHandler")
	}

	sigGasConsumer := opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}

	// maxBypassMinFeeMsgGasUsage is the maximum gas usage per message
	// so that a transaction that contains only message types that can
	// bypass the minimum fee can be accepted with a zero fee.
	// For details, see gaiafeeante.NewFeeDecorator()
	var maxBypassMinFeeMsgGasUsage uint64 = 200_000

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		NewGovPreventSpamDecorator(opts.Codec, opts.GovKeeper),
		migaloofeeante.NewFeeDecorator(opts.BypassMinFeeMsgTypes, opts.GlobalFeeSubspace, opts.StakingSubspace, maxBypassMinFeeMsgGasUsage),

		ante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		ante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(opts.AccountKeeper),
		ante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		ante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		ante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCkeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
