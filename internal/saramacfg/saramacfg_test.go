package saramacfg

import (
	"crypto/tls"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/korableg/bus/sasl"
)

func TestUnit_ApplySASL(t *testing.T) {
	t.Parallel()

	provider := NewMockAccessTokenProvider(gomock.NewController(t))

	tests := []struct {
		name  string
		sasl  *sasl.SASL
		check func(a *assert.Assertions, cfg *sarama.Config)
	}{
		{
			name: "nil",
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.False(cfg.Net.SASL.Enable)
			},
		},
		{
			name: "plaintext",
			sasl: &sasl.SASL{Mechanism: sasl.TypePlaintext, UserName: "user", Password: "pass"},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.True(cfg.Net.SASL.Enable)
				a.Equal(sasl.Mechanism(sasl.TypePlaintext), cfg.Net.SASL.Mechanism)
				a.Equal("user", cfg.Net.SASL.User)
				a.Equal("pass", cfg.Net.SASL.Password)
				a.Nil(cfg.Net.SASL.SCRAMClientGeneratorFunc)
			},
		},
		{
			name: "scram-sha-256",
			sasl: &sasl.SASL{Mechanism: sasl.TypeSCRAMSHA256, UserName: "user", Password: "pass"},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(sasl.Mechanism(sasl.TypeSCRAMSHA256), cfg.Net.SASL.Mechanism)
				a.NotNil(cfg.Net.SASL.SCRAMClientGeneratorFunc)
			},
		},
		{
			name: "scram-sha-512",
			sasl: &sasl.SASL{Mechanism: sasl.TypeSCRAMSHA512, UserName: "user", Password: "pass"},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(sasl.Mechanism(sasl.TypeSCRAMSHA512), cfg.Net.SASL.Mechanism)
				a.NotNil(cfg.Net.SASL.SCRAMClientGeneratorFunc)
			},
		},
		{
			name: "oauth",
			sasl: &sasl.SASL{Mechanism: sasl.TypeOAuth, TokenProvider: provider},
			check: func(a *assert.Assertions, cfg *sarama.Config) {
				a.Equal(sasl.Mechanism(sasl.TypeOAuth), cfg.Net.SASL.Mechanism)
				a.Same(provider, cfg.Net.SASL.TokenProvider)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := sarama.NewConfig()
			ApplySASL(cfg, tt.sasl)

			require.NoError(t, cfg.Validate())
			tt.check(assert.New(t), cfg)
		})
	}
}

func TestUnit_ApplyTLS(t *testing.T) {
	t.Parallel()

	tlsCfg := &tls.Config{ServerName: "kafka.local", MinVersion: tls.VersionTLS12}

	tests := []struct {
		name       string
		tls        *tls.Config
		wantEnable bool
	}{
		{name: "nil", tls: nil, wantEnable: false},
		{name: "config", tls: tlsCfg, wantEnable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := sarama.NewConfig()
			ApplyTLS(cfg, tt.tls)

			require.NoError(t, cfg.Validate())
			assert.Equal(t, tt.wantEnable, cfg.Net.TLS.Enable)

			if tt.wantEnable {
				assert.Same(t, tt.tls, cfg.Net.TLS.Config)
			}
		})
	}
}
