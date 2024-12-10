package fetch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderURLPath(t *testing.T) {
	assert := assert.New(t)

	type Params struct {
		UserID string
		BookID string
	}

	tests := []struct {
		path     string
		data     interface{}
		expected string
	}{
		{
			path:     "/{{UserID}}/{{BookID}}.json",
			data:     map[string]interface{}{"UserID": "123", "BookID": "456"},
			expected: "/123/456.json",
		},
		{
			path:     "/user/{{UserID}}/book/{{BookID}}",
			data:     map[string]interface{}{"UserID": "789", "BookID": "abc"},
			expected: "/user/789/book/abc",
		},
		{
			path:     "/constant/path",
			data:     map[string]interface{}{"UserID": "123", "BookID": "456"},
			expected: "/constant/path",
		},
		{
			path:     "/struct/{{UserID}}/{{BookID}}.json",
			data:     Params{UserID: "321", BookID: "654"},
			expected: "/struct/321/654.json",
		},
		{
			path:     "/user/{{UserID}}/book/{{BookID}}/detail",
			data:     Params{UserID: "987", BookID: "cba"},
			expected: "/user/987/book/cba/detail",
		},
	}

	for _, tt := range tests {
		result, err := RenderURLPath(tt.path, tt.data)
		assert.NoError(err)
		assert.Equal(tt.expected, result)
	}
}
