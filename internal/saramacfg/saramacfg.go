package saramacfg

import (
	"crypto/tls"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"

	"github.com/korableg/bus/sasl"
)

// ApplySASL enables the SASL authentication described by s in cfg. It does nothing if s is nil.
func ApplySASL(cfg *sarama.Config, s *sasl.SASL) {
	if s == nil {
		return
	}

	cfg.Net.SASL.Enable = true
	cfg.Net.SASL.Mechanism = s.Mechanism
	cfg.Net.SASL.User = s.UserName
	cfg.Net.SASL.Password = s.Password

	switch s.Mechanism {
	case sasl.TypeSCRAMSHA256:
		cfg.Net.SASL.SCRAMClientGeneratorFunc = newSCRAMClient(scram.SHA256)
	case sasl.TypeSCRAMSHA512:
		cfg.Net.SASL.SCRAMClientGeneratorFunc = newSCRAMClient(scram.SHA512)
	case sasl.TypeOAuth:
		cfg.Net.SASL.TokenProvider = s.TokenProvider
	}
}

// ApplyTLS enables TLS with tlsCfg in cfg. It does nothing if tlsCfg is nil.
func ApplyTLS(cfg *sarama.Config, tlsCfg *tls.Config) {
	if tlsCfg == nil {
		return
	}

	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = tlsCfg
}
