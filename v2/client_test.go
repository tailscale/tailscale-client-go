package tailscale_test

import (
	_ "embed"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tailscale/tailscale-client-go/v2"
)

func TestErrorData(t *testing.T) {
	t.Parallel()

	t.Run("It should return the data element from a valid error", func(t *testing.T) {
		expected := tailscale.APIError{
			Data: []tailscale.APIErrorData{
				{
					User: "user1@example.com",
					Errors: []string{
						"address \"user2@example.com:400\": want: Accept, got: Drop",
					},
				},
			},
		}

		actual := tailscale.ErrorData(expected)
		assert.EqualValues(t, expected.Data, actual)
	})

	t.Run("It should return an empty slice for any other error", func(t *testing.T) {
		assert.Empty(t, tailscale.ErrorData(io.EOF))
	})
}
