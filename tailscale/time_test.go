package tailscale_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

func TestWrapsStdTime(t *testing.T) {
	expectedTime := tailscale.Time{}
	newTime := time.Time{}
	assert.Equal(t, expectedTime.Time.UTC(), newTime.UTC())
}

func TestFailsToParseInvalidTimestamps(t *testing.T) {
	ti := tailscale.Time{}
	invalidIso8601Date := []byte("\"2022-13-05T17:10:27Z\"")
	assert.Error(t, ti.UnmarshalJSON(invalidIso8601Date))
}

func TestMarshalingTimestamps(t *testing.T) {
	t.Parallel()
	utcMinusFour := time.FixedZone("UTC-4", -60*60*4)

	tt := []struct {
		Name        string
		Expected    time.Time
		TimeContent string
	}{
		{
			Name:        "It should handle empty strings as null-value times",
			Expected:    time.Time{},
			TimeContent: "\"\"",
		},
		{
			Name:        "It should parse timestamp strings",
			Expected:    time.Date(2022, 3, 5, 17, 10, 27, 0, time.UTC),
			TimeContent: "\"2022-03-05T17:10:27Z\"",
		},
		{
			Name:        "It should handle different timezones",
			TimeContent: "\"2006-01-02T15:04:05-04:00\"",
			Expected:    time.Date(2006, 1, 2, 15, 04, 5, 0, utcMinusFour),
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			actual := tailscale.Time{}

			assert.NoError(t, actual.UnmarshalJSON([]byte(tc.TimeContent)))
			assert.Equal(t, tc.Expected.UTC(), actual.Time.UTC())
		})
	}
}

func TestWrapsStdDuration(t *testing.T) {
	expectedDuration := tailscale.Duration{}
	newDuration := time.Duration(0)
	assert.Equal(t, expectedDuration.Duration, newDuration)
}
