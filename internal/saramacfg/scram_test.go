package saramacfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdg-go/scram"
)

func TestUnit_newSCRAMClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hash scram.HashGeneratorFcn
	}{
		{name: "sha-256", hash: scram.SHA256},
		{name: "sha-512", hash: scram.SHA512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			generate := newSCRAMClient(tt.hash)

			c1, c2 := generate(), generate()
			assert.NotSame(t, c1, c2, "every connection needs its own conversation")

			require.NoError(t, c1.Begin("user", "pass", ""))

			first, err := c1.Step("")
			require.NoError(t, err)
			assert.Contains(t, first, "n=user")
			assert.False(t, c1.Done())

			assert.Error(t, c2.Begin("user\a", "pass", ""), "SASLprep rejects the control characters")
		})
	}
}
