package producer

import (
	"github.com/IBM/sarama"

	"github.com/korableg/bus/internal/saramacfg"
)

func newSaramaConfig(o Options) *sarama.Config {
	cfg := sarama.NewConfig()

	cfg.Version = sarama.V3_0_0_0
	cfg.ChannelBufferSize = 1024

	cfg.Net.DialTimeout = o.DialTimeout
	cfg.Net.ReadTimeout = o.ReadTimeout
	cfg.Net.WriteTimeout = o.WriteTimeout
	cfg.Net.MaxOpenRequests = o.MaxOpenRequests

	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	cfg.Producer.Idempotent = o.Idempotence

	cfg.Producer.Flush.MaxMessages = 500000
	cfg.Producer.Flush.Frequency = o.FlushFrequency
	cfg.Producer.Flush.Bytes = o.FlushBytes
	cfg.Producer.Flush.Messages = o.FlushMessages

	cfg.Producer.Retry.Max = o.Retries
	cfg.Producer.Retry.Backoff = o.RetryBackOff

	saramacfg.ApplySASL(cfg, o.SASL)
	saramacfg.ApplyTLS(cfg, o.TLS)

	return cfg
}
