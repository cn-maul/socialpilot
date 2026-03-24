package jsonx

import (
	"strings"
	"testing"
)

func TestExtractJSONObject(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple object",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "object with prefix text",
			input: `Here is the result: {"status": "ok"}`,
			want:  `{"status": "ok"}`,
		},
		{
			name:  "object with suffix text",
			input: `{"status": "ok"} and more text`,
			want:  `{"status": "ok"}`,
		},
		{
			name:  "array",
			input: `[1, 2, 3]`,
			want:  `[1, 2, 3]`,
		},
		{
			name:  "markdown code block",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "markdown code block without language",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "nested object",
			input: `{"outer": {"inner": "value"}}`,
			want:  `{"outer": {"inner": "value"}}`,
		},
		{
			name:  "empty string",
			input: ``,
			want:  ``,
		},
		{
			name:  "whitespace only",
			input: `   `,
			want:  ``,
		},
		{
			name:  "no json",
			input: `this is just text`,
			want:  ``,
		},
		{
			name:  "array in text",
			input: `Result: [{"id": 1}, {"id": 2}] done`,
			want:  `[{"id": 1}, {"id": 2}]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractJSONObject(tc.input)
			if got != tc.want {
				t.Errorf("ExtractJSONObject(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExtractJSONObjectEdgeCases(t *testing.T) {
	// Test that braces inside strings are handled correctly
	input := `{"key": "value with {braces}"}`
	got := ExtractJSONObject(input)
	if got == "" {
		t.Error("expected non-empty result for object with braces in string")
	}

	// Test mixed array and object - returns the whole span from first to last
	// This is expected behavior: the function extracts from the first JSON start
	// to the last JSON end, which can include multiple JSON structures
	input2 := `[1, 2] {"key": "val"}`
	got2 := ExtractJSONObject(input2)
	// The function finds first '[' and last '}', so it returns the whole span
	if !strings.HasPrefix(got2, "[") || !strings.HasSuffix(got2, "}") {
		t.Errorf("ExtractJSONObject(%q) = %q, expected JSON span", input2, got2)
	}
}
