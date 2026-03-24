package llm

import (
	"testing"
)

func TestChatCompletionsURL(t *testing.T) {
	cases := []struct {
		name string
		base string
		want string
	}{
		{
			name: "openai root adds v1",
			base: "https://api.openai.com",
			want: "https://api.openai.com/v1/chat/completions",
		},
		{
			name: "v1 path appends endpoint",
			base: "https://example.com/v1",
			want: "https://example.com/v1/chat/completions",
		},
		{
			name: "custom provider path keeps path",
			base: "https://qianfan.baidubce.com/v2/coding/v1",
			want: "https://qianfan.baidubce.com/v2/coding/v1/chat/completions",
		},
		{
			name: "already full endpoint",
			base: "https://example.com/v2/coding/v1/chat/completions",
			want: "https://example.com/v2/coding/v1/chat/completions",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.base, "k", "m", 0)
			got := c.chatCompletionsURL()
			if got != tc.want {
				t.Fatalf("chatCompletionsURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUseProxyFromEnv(t *testing.T) {
	t.Setenv("SOCIALPILOT_USE_PROXY", "1")
	if !useProxyFromEnv() {
		t.Fatalf("expected proxy enabled")
	}

	t.Setenv("SOCIALPILOT_USE_PROXY", "false")
	if useProxyFromEnv() {
		t.Fatalf("expected proxy disabled")
	}
}
