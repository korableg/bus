package sasl

import "github.com/IBM/sarama"

// Supported SASL mechanisms.
const (
	// TypeOAuth is OAUTHBEARER, the token comes from SASL.TokenProvider.
	TypeOAuth = sarama.SASLTypeOAuth
	// TypePlaintext is PLAIN with the user name and the password.
	TypePlaintext = sarama.SASLTypePlaintext
	// TypeSCRAMSHA256 is SCRAM-SHA-256 with the user name and the password.
	TypeSCRAMSHA256 = sarama.SASLTypeSCRAMSHA256
	// TypeSCRAMSHA512 is SCRAM-SHA-512 with the user name and the password.
	TypeSCRAMSHA512 = sarama.SASLTypeSCRAMSHA512
)

type (
	// Mechanism is the SASL mechanism, one of the Type* constants.
	Mechanism = sarama.SASLMechanism

	// TokenProvider provides the access tokens for the OAuth (OAUTHBEARER) mechanism.
	// The implementation should reuse the token and refresh it periodically.
	TokenProvider = sarama.AccessTokenProvider

	// AccessToken is the access token returned by TokenProvider.
	AccessToken = sarama.AccessToken

	// SASL holds the authentication settings. UserName and Password are used by PLAIN and SCRAM,
	// TokenProvider by OAUTHBEARER.
	SASL struct {
		Mechanism     Mechanism
		UserName      string
		Password      string
		TokenProvider TokenProvider
	}
)
