package consumer

import (
	"bytes"
	"context"
	"crypto/tls"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/korableg/bus/sasl"
)

func TestUnit_newOptions(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		o := newOptions()

		assert.Equal(t, InitialOffsetLatest, o.InitialOffset)
		assert.Equal(t, 500*time.Millisecond, o.FetchMaxWait)
		assert.Equal(t, int32(1), o.FetchMinBytes)
		assert.Equal(t, 20, o.CommitCount)
		assert.Equal(t, time.Second, o.CommitDuration)
		assert.Equal(t, 5*time.Second, o.DialTimeout)
		assert.Equal(t, 10*time.Second, o.ReadTimeout)
		assert.Equal(t, 5*time.Second, o.WriteTimeout)
		assert.Equal(t, slog.Default(), o.Logger)
		assert.Nil(t, o.SASL)
		assert.Nil(t, o.TLS)

		ctx := context.Background()
		assert.Equal(t, ctx, o.TraceIDCtx(ctx, "trace"))
	})

	t.Run("options", func(t *testing.T) {
		t.Parallel()

		var (
			l      = slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
			tlsCfg = &tls.Config{ServerName: "kafka"}
			o      = newOptions(
				WithLogger(l),
				WithCommitCount(5),
				WithCommitDuration(6),
				WithEarliestOffsetReset(),
				WithFetchMaxWait(7),
				WithFetchMinBytes(8),
				WithDialTimeout(15*time.Second),
				WithReadTimeout(20*time.Second),
				WithWriteTimeout(25*time.Second),
				WithTraceIDCtxFunc(withTrace),
				WithSASL(sasl.TypePlaintext, "user", "pass"),
				WithTLS(tlsCfg),
			)
		)

		assert.Equal(t, InitialOffsetEarliest, o.InitialOffset)
		assert.Equal(t, time.Duration(7), o.FetchMaxWait)
		assert.Equal(t, int32(8), o.FetchMinBytes)
		assert.Equal(t, 5, o.CommitCount)
		assert.Equal(t, time.Duration(6), o.CommitDuration)
		assert.Equal(t, 15*time.Second, o.DialTimeout)
		assert.Equal(t, 20*time.Second, o.ReadTimeout)
		assert.Equal(t, 25*time.Second, o.WriteTimeout)
		assert.Same(t, l, o.Logger)
		assert.Equal(t, "trace", o.TraceIDCtx(context.Background(), "trace").Value(traceKey{}))
		assert.Equal(t, &sasl.SASL{Mechanism: sasl.TypePlaintext, UserName: "user", Password: "pass"}, o.SASL)
		assert.Same(t, tlsCfg, o.TLS)
	})

	t.Run("nil values", func(t *testing.T) {
		t.Parallel()

		o := newOptions(WithLogger(nil), WithTraceIDCtxFunc(nil), WithTLS(nil))

		assert.Equal(t, slog.Default(), o.Logger)
		require.NotNil(t, o.TraceIDCtx)
		assert.NotNil(t, o.TLS, "a nil TLS config means the default one")
	})
}
