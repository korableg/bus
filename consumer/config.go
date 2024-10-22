package consumer

import (
	"github.com/IBM/sarama"
)

func newSaramaConfig(o Options) *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.ChannelBufferSize = 1024
	cfg.Version = sarama.V3_0_0_0

	cfg.Net.DialTimeout = o.DialTimeout
	cfg.Net.ReadTimeout = o.ReadTimeout
	cfg.Net.WriteTimeout = o.WriteTimeout

	cfg.Consumer.Offsets.Initial = o.InitialOffset
	cfg.Consumer.Offsets.AutoCommit.Enable = false
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Fetch.Min = o.FetchMinBytes
	cfg.Consumer.MaxWaitTime = o.FetchMaxWait
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategySticky(),
		sarama.NewBalanceStrategyRoundRobin(),
		sarama.NewBalanceStrategyRange(),
	}

	if o.SASL != nil {
		cfg.Net.SASL.Enable = true
		cfg.Net.SASL.Mechanism = o.SASL.Mechanism
		cfg.Net.SASL.User = o.SASL.UserName
		cfg.Net.SASL.Password = o.SASL.Password
	}

	return cfg
}
