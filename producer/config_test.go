package producer

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
				a.Equal(sarama.V3_0_0_0, cfg.Version)
				a.Equal(1024, cfg.ChannelBufferSize)
				a.Equal(5*time.Second, cfg.Net.DialTimeout)
				a.Equal(10*time.Second, cfg.Net.ReadTimeout)
				a.Equal(5*time.Second, cfg.Net.WriteTimeout)
				a.Equal(1, cfg.Net.MaxOpenRequests)
				a.Equal(sarama.WaitForAll, cfg.Producer.RequiredAcks)
				a.True(cfg.Producer.Return.Successes)
				a.True(cfg.Producer.Return.Errors)
				a.True(cfg.Producer.Idempotent)
				a.Equal(10, cfg.Producer.Retry.Max)
				a.Equal(300*time.Millisecond, cfg.Producer.Retry.Backoff)
				a.False(cfg.Net.SASL.Enable)
				a.False(cfg.Net.TLS.Enable)
			},
		},
		{
			name: "without idempotence",
			opts: []Option{WithoutIdempotence(), WithMaxOpenRequests(5), WithFlushParams(time.Second, 2, 3)},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.False(cfg.Producer.Idempotent)
				a.Equal(5, cfg.Net.MaxOpenRequests)
				a.Equal(time.Second, cfg.Producer.Flush.Frequency)
				a.Equal(2, cfg.Producer.Flush.Bytes)
				a.Equal(3, cfg.Producer.Flush.Messages)
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
			opts: []Option{WithSASL(sasl.TypeSCRAMSHA512, "user", "pass")},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(sasl.Mechanism(sasl.TypeSCRAMSHA512), cfg.Net.SASL.Mechanism)
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
