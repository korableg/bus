package producer

import (
	"github.com/IBM/sarama"

	"github.com/korableg/bus/sasl"
)

func newSaramaConfig(sasl *sasl.SASL, o Options) *sarama.Config {
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

	if sasl != nil {
		cfg.Net.SASL.Enable = true
		cfg.Net.SASL.Mechanism = sasl.Mechanism
		cfg.Net.SASL.User = sasl.UserName
		cfg.Net.SASL.Password = sasl.Password
	}

	return cfg
}
