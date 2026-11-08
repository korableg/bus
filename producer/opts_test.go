package producer

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

		assert.True(t, o.Idempotence)
		assert.Equal(t, 1, o.MaxOpenRequests)
		assert.Zero(t, o.FlushFrequency)
		assert.Zero(t, o.FlushBytes)
		assert.Zero(t, o.FlushMessages)
		assert.Equal(t, 10, o.Retries)
		assert.Equal(t, 300*time.Millisecond, o.RetryBackOff)
		assert.Equal(t, 5*time.Second, o.DialTimeout)
		assert.Equal(t, 10*time.Second, o.ReadTimeout)
		assert.Equal(t, 5*time.Second, o.WriteTimeout)
		assert.Equal(t, slog.Default(), o.Logger)
		assert.Empty(t, o.TraceID(context.Background()))
		assert.Nil(t, o.SASL)
		assert.Nil(t, o.TLS)
		assert.Nil(t, o.ErrorHandler)
	})

	t.Run("options", func(t *testing.T) {
		t.Parallel()

		var (
			l          = slog.New(slog.NewJSONHandler(bytes.NewBuffer(nil), nil))
			tlsCfg     = &tls.Config{ServerName: "kafka"}
			errHandled bool
			o          = newOptions(
				WithoutIdempotence(),
				WithMaxOpenRequests(5),
				WithFlushParams(1, 2, 3),
				WithRetries(3),
				WithRetryBackOff(4),
				WithDialTimeout(15*time.Second),
				WithReadTimeout(20*time.Second),
				WithWriteTimeout(25*time.Second),
				WithLogger(l),
				WithTraceID(func(context.Context) string { return "trace-id" }),
				WithSASL(sasl.TypePlaintext, "user", "pass"),
				WithErrorHandler(func(error) { errHandled = true }),
				WithTLS(tlsCfg),
			)
		)

		assert.False(t, o.Idempotence)
		assert.Equal(t, 5, o.MaxOpenRequests)
		assert.Equal(t, time.Duration(1), o.FlushFrequency)
		assert.Equal(t, 2, o.FlushBytes)
		assert.Equal(t, 3, o.FlushMessages)
		assert.Equal(t, 3, o.Retries)
		assert.Equal(t, time.Duration(4), o.RetryBackOff)
		assert.Equal(t, 15*time.Second, o.DialTimeout)
		assert.Equal(t, 20*time.Second, o.ReadTimeout)
		assert.Equal(t, 25*time.Second, o.WriteTimeout)
		assert.Same(t, l, o.Logger)
		assert.Equal(t, "trace-id", o.TraceID(context.Background()))
		assert.Equal(t, &sasl.SASL{Mechanism: sasl.TypePlaintext, UserName: "user", Password: "pass"}, o.SASL)
		assert.Same(t, tlsCfg, o.TLS)

		o.ErrorHandler(nil)
		assert.True(t, errHandled)
	})

	t.Run("nil values", func(t *testing.T) {
		t.Parallel()

		o := newOptions(WithLogger(nil), WithTraceID(nil), WithTLS(nil))

		assert.Equal(t, slog.Default(), o.Logger)
		require.NotNil(t, o.TraceID)
		assert.Empty(t, o.TraceID(context.Background()))
		assert.NotNil(t, o.TLS, "a nil TLS config means the default one")
	})
}
