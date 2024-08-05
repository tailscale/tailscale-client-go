package tsclient

import (
	_ "embed"
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorData(t *testing.T) {
	t.Parallel()

	t.Run("It should return the data element from a valid error", func(t *testing.T) {
		expected := APIError{
			Data: []APIErrorData{
				{
					User: "user1@example.com",
					Errors: []string{
						"address \"user2@example.com:400\": want: Accept, got: Drop",
					},
				},
			},
		}

		actual := ErrorData(expected)
		assert.EqualValues(t, expected.Data, actual)
	})

	t.Run("It should return an empty slice for any other error", func(t *testing.T) {
		assert.Empty(t, ErrorData(io.EOF))
	})
}

func Test_BuildTailnetURL(t *testing.T) {
	t.Parallel()

	base, err := url.Parse("http://example.com")
	require.NoError(t, err)

	c := &Client{
		BaseURL: base,
		Tailnet: "tn/with/slashes",
	}
	actual := c.buildTailnetURL("component/with/slashes")
	expected, err := url.Parse("http://example.com/api/v2/tailnet/tn%2Fwith%2Fslashes/component%2Fwith%2Fslashes")
	require.NoError(t, err)
	assert.EqualValues(t, expected.String(), actual.String())
}
