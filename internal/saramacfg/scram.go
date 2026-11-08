package saramacfg

import (
	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

type scramClient struct {
	hash scram.HashGeneratorFcn

	conv *scram.ClientConversation
}

func newSCRAMClient(hash scram.HashGeneratorFcn) func() sarama.SCRAMClient {
	return func() sarama.SCRAMClient {
		return &scramClient{hash: hash}
	}
}

func (c *scramClient) Begin(userName, password, authzID string) error {
	client, err := c.hash.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}

	c.conv = client.NewConversation()

	return nil
}

func (c *scramClient) Step(challenge string) (string, error) {
	return c.conv.Step(challenge)
}

func (c *scramClient) Done() bool {
	return c.conv.Done()
}
