package producer

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/korableg/bus/sasl"
)

func TestUnit_Options_Producer(t *testing.T) {
	t.Parallel()

	var (
		l           = slog.New(slog.NewJSONHandler(bytes.NewBuffer(nil), nil))
		traceIDFunc = func(context.Context) string {
			return "trace-id-func"
		}
		o = newOptions(
			WithoutIdempotence(),
			WithMaxOpenRequests(5),
			WithFlushParams(1, 2, 3),
			WithRetries(3),
			WithRetryBackOff(4),
			WithDialTimeout(15*time.Second),
			WithReadTimeout(20*time.Second),
			WithWriteTimeout(25*time.Second),
			WithLogger(l),
			WithTraceID(traceIDFunc),
			WithSASL(sasl.TypeOAuth, "dmitry", "titov"),
		)
	)

	assert.False(t, o.Idempotence)
	assert.Equal(t, o.MaxOpenRequests, 5)

	assert.Equal(t, o.FlushFrequency, time.Duration(1))
	assert.Equal(t, o.FlushBytes, 2)
	assert.Equal(t, o.FlushMessages, 3)

	assert.Equal(t, o.Retries, 3)
	assert.Equal(t, o.RetryBackOff, time.Duration(4))

	assert.Equal(t, o.DialTimeout, 15*time.Second)
	assert.Equal(t, o.ReadTimeout, 20*time.Second)
	assert.Equal(t, o.WriteTimeout, 25*time.Second)

	assert.Equal(t, o.Logger, l)
	assert.Equal(t, o.TraceID(context.Background()), "trace-id-func")
	assert.Equal(t, o.SASL, &sasl.SASL{
		Mechanism: sasl.TypeOAuth,
		UserName:  "dmitry",
		Password:  "titov",
	})
}
