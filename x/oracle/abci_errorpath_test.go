package oracle_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/classic-terra/core/v4/x/oracle"
	"github.com/classic-terra/core/v4/x/oracle/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
)

// capturingLogger is a minimal log.Logger that records error-level messages.
type capturingLogger struct {
	mu     sync.Mutex
	errors []string
}

func (l *capturingLogger) Info(string, ...any)  {}
func (l *capturingLogger) Warn(string, ...any)  {}
func (l *capturingLogger) Debug(string, ...any) {}

func (l *capturingLogger) With(keyVals ...any) log.Logger { return l }

func (l *capturingLogger) Impl() any { return l }

func (l *capturingLogger) Error(msg string, keyVals ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := msg
	for _, kv := range keyVals {
		if err, ok := kv.(error); ok {
			entry += " error=" + err.Error()
		}
	}
	l.errors = append(l.errors, entry)
}

func (l *capturingLogger) containsError(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, msg := range l.errors {
		if strings.Contains(msg, substr) {
			return true
		}
	}
	return false
}

// stakingKeeperWrapper wraps the real staking keeper and can inject failures
// into individual methods, implementing the oracle module's StakingKeeper
// interface.
type stakingKeeperWrapper struct {
	*stakingkeeper.Keeper

	failMaxValidators bool
	failPowerIterator bool
}

var _ types.StakingKeeper = stakingKeeperWrapper{}

func (w stakingKeeperWrapper) MaxValidators(ctx context.Context) (uint32, error) {
	if w.failMaxValidators {
		return 0, errors.New("injected MaxValidators failure")
	}
	return w.Keeper.MaxValidators(ctx)
}

func (w stakingKeeperWrapper) ValidatorsPowerStoreIterator(ctx context.Context) (storetypes.Iterator, error) {
	if w.failPowerIterator {
		return nil, errors.New("injected ValidatorsPowerStoreIterator failure")
	}
	return w.Keeper.ValidatorsPowerStoreIterator(ctx)
}

// TestEndBlockerStakingKeeperErrorLogging verifies that the EndBlocker logs an
// error (instead of returning silently) when the staking keeper fails, and
// that such failures do not panic.
func TestEndBlockerStakingKeeperErrorLogging(t *testing.T) {
	input, _ := setup(t)

	t.Run("MaxValidators error is logged and does not panic", func(t *testing.T) {
		logger := &capturingLogger{}
		ctx := input.Ctx.WithLogger(logger)

		input.OracleKeeper.StakingKeeper = stakingKeeperWrapper{
			Keeper:            input.StakingKeeper,
			failMaxValidators: true,
		}

		require.NotPanics(t, func() {
			oracle.EndBlocker(ctx, input.OracleKeeper)
		})
		require.True(t, logger.containsError("max validators"),
			"expected an error log when MaxValidators fails")
	})

	t.Run("ValidatorsPowerStoreIterator error is logged and does not panic", func(t *testing.T) {
		logger := &capturingLogger{}
		ctx := input.Ctx.WithLogger(logger)

		input.OracleKeeper.StakingKeeper = stakingKeeperWrapper{
			Keeper:           input.StakingKeeper,
			failPowerIterator: true,
		}

		require.NotPanics(t, func() {
			oracle.EndBlocker(ctx, input.OracleKeeper)
		})
		require.True(t, logger.containsError("validator power store"),
			"expected an error log when ValidatorsPowerStoreIterator fails")
	})
}
