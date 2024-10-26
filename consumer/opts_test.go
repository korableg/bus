package consumer

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/korableg/bus/sasl"
)

func TestUnit_Options(t *testing.T) {
	t.Parallel()

	var (
		l = slog.New(slog.NewTextHandler(bytes.NewBufferString(""), nil))
		o = newOptions(
			WithLogger(l),
			WithCommitCount(5),
			WithCommitDuration(6),
			WithEarliestOffsetReset(),
			WithFetchMaxWait(7),
			WithFetchMinBytes(8),
			WithDialTimeout(15*time.Second),
			WithReadTimeout(20*time.Second),
			WithWriteTimeout(25*time.Second),
			WithTraceIDCtxFunc(func(ctx context.Context, s string) context.Context {
				return context.WithValue(ctx, "trace-id", "test_trace_id") //nolint: staticcheck
			}),
			WithSASL(sasl.TypePlaintext, "dmitry", "titov"),
		)
	)

	assert.Equal(t, o.InitialOffset, InitialOffsetEarliest)

	assert.Equal(t, o.FetchMaxWait, time.Duration(7))
	assert.Equal(t, o.FetchMinBytes, int32(8))

	assert.Equal(t, o.CommitDuration, time.Duration(6))
	assert.Equal(t, o.CommitCount, 5)

	assert.Equal(t, o.DialTimeout, 15*time.Second)
	assert.Equal(t, o.ReadTimeout, 20*time.Second)
	assert.Equal(t, o.WriteTimeout, 25*time.Second)

	assert.Equal(t, o.Logger, l)

	ctx := o.TraceIDCtx(context.Background(), "")
	assert.Equal(t, ctx.Value("trace-id").(string), "test_trace_id")

	assert.Equal(t, o.SASL, &sasl.SASL{
		Mechanism: sasl.TypePlaintext,
		UserName:  "dmitry",
		Password:  "titov",
	})
}
