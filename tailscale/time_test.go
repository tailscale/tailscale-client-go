package tailscale_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tailscale/tailscale-client-go/tailscale"
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

func TestDurationUnmarshal(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Content  string
		Expected tailscale.Duration
	}{
		{
			Name:     "It should handle empty string as zero value",
			Content:  `""`,
			Expected: tailscale.Duration(0),
		},
		{
			Name:     "It should handle null as zero value",
			Content:  `null`,
			Expected: tailscale.Duration(0),
		},
		{
			Name:     "It should handle 0s as zero value",
			Content:  `"0s"`,
			Expected: tailscale.Duration(0),
		},
		{
			Name:     "It should parse duration strings",
			Content:  `"15s"`,
			Expected: tailscale.Duration(15 * time.Second),
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			var actual tailscale.Duration

			assert.NoError(t, json.Unmarshal([]byte(tc.Content), &actual))
			assert.Equal(t, tc.Expected, actual)
		})
	}
}

func TestDurationMarshal(t *testing.T) {
	t.Parallel()

	tt := []struct {
		Name     string
		Content  any
		Expected string
	}{
		{
			Name:     "It should marshal zero duration as 0s",
			Content:  struct{ D tailscale.Duration }{tailscale.Duration(0)},
			Expected: `{"D":"0s"}`,
		},
		{
			Name: "It should not marshal zero duration if omitempty",
			Content: struct {
				D tailscale.Duration `json:"d,omitempty"`
			}{tailscale.Duration(0)},
			Expected: `{}`,
		},
		{
			Name:     "It should marshal duration correctly",
			Content:  struct{ D tailscale.Duration }{tailscale.Duration(15 * time.Second)},
			Expected: `{"D":"15s"}`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			actual, err := json.Marshal(tc.Content)

			assert.NoError(t, err)
			assert.Equal(t, tc.Expected, string(actual))
		})
	}
}
