// Package saramacfg applies the connection settings (SASL, TLS) shared by the producer
// and the consumer to the sarama config.
package saramacfg

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=sarama_mock_test.go -package=saramacfg github.com/IBM/sarama AccessTokenProvider
