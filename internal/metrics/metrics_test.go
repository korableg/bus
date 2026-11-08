package metrics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnit_ErrToLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil error", err: nil, want: "false"},
		{name: "error", err: errors.New("err"), want: "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, ErrToLabel(tt.err))
		})
	}
}
