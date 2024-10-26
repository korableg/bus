package sasl

import "github.com/IBM/sarama"

const (
	TypeOAuth       = sarama.SASLTypeOAuth
	TypePlaintext   = sarama.SASLTypePlaintext
	TypeSCRAMSHA256 = sarama.SASLTypeSCRAMSHA256
	TypeSCRAMSHA512 = sarama.SASLTypeSCRAMSHA512
)

type (
	Mechanism = sarama.SASLMechanism

	SASL struct {
		Mechanism Mechanism
		UserName  string
		Password  string
	}
)
