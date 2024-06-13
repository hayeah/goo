package fetch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type UserData struct {
	Name string
	Age  int
}

func TestRenderJSON(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		template string
		data     interface{}
		expected string
	}{
		{
			template: `{"user_id": {{UserID}}, "book_id": {{BookID}}}`,
			data:     map[string]interface{}{"UserID": "123", "BookID": "456"},
			expected: `{"user_id": "123", "book_id": "456"}`,
		},

		{
			template: `{"name": {{Name}}, "age": {{Age}}}`,
			data:     UserData{Name: "Alice", Age: 25},
			expected: `{"name": "Alice", "age": 25}`,
		},

		{
			template: `
			{"list": [
				{{#.}} 
				{ "name": {{Name}} }, 
				{{/.}}
			]} `,
			data: []UserData{
				{Name: "Alice", Age: 25},
				{Name: "Bob", Age: 30},
			},
			expected: `{"list": [{"name": "Alice"}, {"name": "Bob"}]}`,
		},
	}

	for _, tt := range tests {
		result, err := RenderJSON(tt.template, tt.data)
		assert.NoError(err)
		assert.JSONEq(tt.expected, string(result))
	}
}

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
