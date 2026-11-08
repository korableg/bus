package consumer

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/korableg/bus/sasl"
)

func TestUnit_newSaramaConfig(t *testing.T) {
	t.Parallel()

	var (
		tlsCfg   = &tls.Config{ServerName: "kafka.local", MinVersion: tls.VersionTLS12}
		provider = NewMockAccessTokenProvider(gomock.NewController(t))
	)

	tests := []struct {
		name  string
		opts  []Option
		check func(a *assert.Assertions, cfg *sarama.Config)
	}{
		{
			name: "defaults",
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(1024, cfg.ChannelBufferSize)
				a.Equal(sarama.V3_0_0_0, cfg.Version)
				a.Equal(5*time.Second, cfg.Net.DialTimeout)
				a.Equal(10*time.Second, cfg.Net.ReadTimeout)
				a.Equal(5*time.Second, cfg.Net.WriteTimeout)
				a.Equal(InitialOffsetLatest, cfg.Consumer.Offsets.Initial)
				a.False(cfg.Consumer.Offsets.AutoCommit.Enable)
				a.True(cfg.Consumer.Return.Errors)
				a.Equal(int32(1), cfg.Consumer.Fetch.Min)
				a.Equal(500*time.Millisecond, cfg.Consumer.MaxWaitTime)
				a.Len(cfg.Consumer.Group.Rebalance.GroupStrategies, 3)
				a.False(cfg.Net.SASL.Enable)
				a.False(cfg.Net.TLS.Enable)
			},
		},
		{
			name: "fetch and offset",
			opts: []Option{WithEarliestOffsetReset(), WithFetchMinBytes(1024), WithFetchMaxWait(time.Second)},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(InitialOffsetEarliest, cfg.Consumer.Offsets.Initial)
				a.Equal(int32(1024), cfg.Consumer.Fetch.Min)
				a.Equal(time.Second, cfg.Consumer.MaxWaitTime)
			},
		},
		{
			name: "sasl plaintext",
			opts: []Option{WithSASL(sasl.TypePlaintext, "user", "pass")},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.True(cfg.Net.SASL.Enable)
				a.Equal(sasl.Mechanism(sasl.TypePlaintext), cfg.Net.SASL.Mechanism)
				a.Equal("user", cfg.Net.SASL.User)
				a.Equal("pass", cfg.Net.SASL.Password)
			},
		},
		{
			name: "sasl scram",
			opts: []Option{WithSASL(sasl.TypeSCRAMSHA256, "user", "pass")},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(sasl.Mechanism(sasl.TypeSCRAMSHA256), cfg.Net.SASL.Mechanism)
				a.NotNil(cfg.Net.SASL.SCRAMClientGeneratorFunc)
			},
		},
		{
			name: "sasl oauth",
			opts: []Option{WithSASLOAuth(provider)},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(sasl.Mechanism(sasl.TypeOAuth), cfg.Net.SASL.Mechanism)
				a.Same(provider, cfg.Net.SASL.TokenProvider)
			},
		},
		{
			name: "tls",
			opts: []Option{WithTLS(tlsCfg)},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.True(cfg.Net.TLS.Enable)
				a.Same(tlsCfg, cfg.Net.TLS.Config)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newSaramaConfig(newOptions(tt.opts...))

			require.NoError(t, cfg.Validate())
			tt.check(assert.New(t), cfg)
		})
	}
}
