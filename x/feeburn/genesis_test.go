package feeburn_test

import (
	"testing"

	keepertest "github.com/ChihuahuaChain/chihuahua/testutil/keeper"
	"github.com/ChihuahuaChain/chihuahua/testutil/nullify"

	"github.com/White-Whale-Defi-Platform/migaloo-chain/v3/x/feeburn"
	"github.com/White-Whale-Defi-Platform/migaloo-chain/v3/x/feeburn/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.FeeburnKeeper(t)
	feeburn.InitGenesis(ctx, *k, genesisState)
	got := feeburn.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
