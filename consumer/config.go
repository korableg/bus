package consumer

import (
	"time"

	"github.com/IBM/sarama"
	"gitlab.services.mts.ru/scs-data-platform/backend/modules/protobus"
)

type (
	Sasl struct {
		Mechanism string `env:"mechanism"`
		UserName  string `env:"username"`
		Password  string `env:"password"`
	}

	GroupConfig struct {
		Brokers string `env:"brokers"`
		Group   string `env:"group"`
		Sasl    Sasl   `env:"sasl"`
	}

	FacadeConfig struct {
		Brokers string `env:"brokers"`
		Sasl    Sasl   `env:"sasl"`
	}
)

func NewGroupConfig() GroupConfig {
	return GroupConfig{
		Sasl: Sasl{
			Mechanism: protobus.SecuritySaslPlain,
		},
	}
}

func NewFacadeConfig() FacadeConfig {
	return FacadeConfig{
		Sasl: Sasl{
			Mechanism: protobus.SecuritySaslPlain,
		},
	}
}

func newSaramaConfig(sasl Sasl, o Options) *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.ChannelBufferSize = 1024
	cfg.Version = sarama.V3_0_0_0

	cfg.Net.DialTimeout = 5 * time.Second
	cfg.Net.ReadTimeout = 10 * time.Second
	cfg.Net.WriteTimeout = 5 * time.Second

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

	if sasl.UserName != "" {
		cfg.Net.SASL.Enable = true
		cfg.Net.SASL.Mechanism = sarama.SASLMechanism(sasl.Mechanism)
		cfg.Net.SASL.User = sasl.UserName
		cfg.Net.SASL.Password = sasl.Password
	}

	return cfg
}
