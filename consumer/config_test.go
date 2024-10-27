package consumer

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"

	"github.com/korableg/bus/sasl"
)

func TestUnit_newSaramaConfig(t *testing.T) {
	t.Parallel()

	cfg := newSaramaConfig(newOptions(WithSASL(sasl.TypePlaintext, "dmitry", "titov")))

	assert.Equal(t, cfg.ChannelBufferSize, 1024)
	assert.Equal(t, cfg.Version, sarama.V3_0_0_0)
	assert.Equal(t, cfg.Net.DialTimeout, 5*time.Second)
	assert.Equal(t, cfg.Net.ReadTimeout, 10*time.Second)
	assert.Equal(t, cfg.Net.WriteTimeout, 5*time.Second)
	assert.Equal(t, cfg.Consumer.Offsets.Initial, InitialOffsetLatest)
	assert.Equal(t, cfg.Consumer.Offsets.AutoCommit.Enable, false)
	assert.Equal(t, cfg.Consumer.Return.Errors, true)
	assert.Equal(t, cfg.Consumer.Fetch.Min, int32(1))
	assert.Equal(t, cfg.Consumer.MaxWaitTime, 500*time.Millisecond)
	assert.Len(t, cfg.Consumer.Group.Rebalance.GroupStrategies, 3)
	assert.Equal(t, cfg.Net.SASL.Enable, true)
	assert.Equal(t, cfg.Net.SASL.Mechanism, sarama.SASLMechanism("PLAIN"))
	assert.Equal(t, cfg.Net.SASL.User, "dmitry")
	assert.Equal(t, cfg.Net.SASL.Password, "titov")
}
