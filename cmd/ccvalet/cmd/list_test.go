package cmd

import "testing"

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		maxLen int
		want   string
	}{
		{
			name:   "short path no truncation",
			path:   "/home/user",
			maxLen: 20,
			want:   "/home/user",
		},
		{
			name:   "exact maxLen no truncation",
			path:   "/home/user",
			maxLen: 10,
			want:   "/home/user",
		},
		{
			name:   "long path truncated with prefix ellipsis",
			path:   "/home/user/projects/my-very-long-project-name",
			maxLen: 20,
			want:   "...long-project-name",
		},
		{
			name:   "empty string",
			path:   "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncatePath(tt.path, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncatePath(%q, %d) = %q, want %q", tt.path, tt.maxLen, got, tt.want)
			}
			if tt.maxLen > 0 && len(got) > tt.maxLen {
				t.Errorf("truncatePath(%q, %d) returned %d chars, exceeds maxLen %d", tt.path, tt.maxLen, len(got), tt.maxLen)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "short string no truncation",
			s:      "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact maxLen no truncation",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated with suffix ellipsis",
			s:      "this is a very long error message",
			maxLen: 15,
			want:   "this is a ve...",
		},
		{
			name:   "maxLen equals 3 no ellipsis",
			s:      "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "maxLen equals 2 no ellipsis",
			s:      "hello",
			maxLen: 2,
			want:   "he",
		},
		{
			name:   "maxLen equals 1",
			s:      "hello",
			maxLen: 1,
			want:   "h",
		},
		{
			name:   "maxLen equals 0",
			s:      "hello",
			maxLen: 0,
			want:   "",
		},
		{
			name:   "empty string",
			s:      "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateStr(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
			if tt.maxLen > 0 && len(got) > tt.maxLen {
				t.Errorf("truncateStr(%q, %d) returned %d chars, exceeds maxLen %d", tt.s, tt.maxLen, len(got), tt.maxLen)
			}
		})
	}
}
