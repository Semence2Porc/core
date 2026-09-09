package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"github.com/classic-terra/core/v4/x/tax/keeper"
	"github.com/classic-terra/core/v4/x/tax/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// TestIsReverseCharge covers the reverse-charge context flag handling.
//
// Regression test: the flag is only set by the ante handler. On contexts that
// did not flow through the ante handler (queries, genesis import/export,
// external keeper calls) the flag is absent from the context, which must not
// panic but simply report "no reverse charge".
func TestIsReverseCharge(t *testing.T) {
	tk := keeper.Keeper{}
	baseCtx := sdk.NewContext(nil, tmproto.Header{}, false, log.NewNopLogger())

	t.Run("absent flag does not panic", func(t *testing.T) {
		require.NotPanics(t, func() {
			require.False(t, tk.IsReverseCharge(baseCtx, false))
		})
	})

	t.Run("explicit false returns false", func(t *testing.T) {
		ctx := baseCtx.WithValue(types.ContextKeyTaxReverseCharge, false)
		require.False(t, tk.IsReverseCharge(ctx, false))
	})

	t.Run("explicit true returns true", func(t *testing.T) {
		ctx := baseCtx.WithValue(types.ContextKeyTaxReverseCharge, true)
		require.True(t, tk.IsReverseCharge(ctx, false))
	})

	t.Run("wrong type flag does not panic", func(t *testing.T) {
		ctx := baseCtx.WithValue(types.ContextKeyTaxReverseCharge, "invalid")
		require.NotPanics(t, func() {
			require.False(t, tk.IsReverseCharge(ctx, false))
		})
	})

	t.Run("emits no-reverse-charge event when requested", func(t *testing.T) {
		em := sdk.NewEventManager()
		ctx := baseCtx.WithEventManager(em).WithValue(types.ContextKeyTaxReverseCharge, false)

		tk.IsReverseCharge(ctx, true)

		events := em.Events()
		require.Len(t, events, 1)
		require.Equal(t, types.EventTypeTax, events[0].Type)
	})

	t.Run("does not emit event when not requested", func(t *testing.T) {
		em := sdk.NewEventManager()
		ctx := baseCtx.WithEventManager(em).WithValue(types.ContextKeyTaxReverseCharge, false)

		tk.IsReverseCharge(ctx, false)

		require.Empty(t, em.Events())
	})
}
