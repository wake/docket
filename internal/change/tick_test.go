package change

import "testing"

func TestTick(t *testing.T) {
	tests := []struct {
		name, in string
		lines    []int
		want     string
		wantErr  bool
	}{
		{"lf", "# P\n- [ ] a\n- [ ] b\n", []int{3}, "# P\n- [ ] a\n- [x] b\n", false},
		{"bom crlf", "\xef\xbb\xbf# P\r\n- [ ] a\r\n  * [ ] b\r\n", []int{2, 3}, "\xef\xbb\xbf# P\r\n- [x] a\r\n  * [x] b\r\n", false},
		{"lone cr", "# P\r- [ ] a\r- [ ] b\r", []int{2}, "# P\r- [x] a\r- [ ] b\r", false},
		{"twice", "- [ ] a\n", []int{1, 1}, "- [x] a\n", false},
		{"line 0", "- [ ] a\n", []int{0}, "", true},
		{"past end", "- [ ] a\n", []int{9}, "", true},
		{"already ticked", "- [x] a\n", []int{1}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Tick([]byte(tt.in), tt.lines)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if err != nil && got != nil {
				t.Fatalf("result with error: %q", got)
			}
			if err == nil && string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTickPartialFailureGivesNoResult(t *testing.T) {
	if got, err := Tick([]byte("- [ ] a\n- [x] b\n"), []int{1, 2}); err == nil || got != nil {
		t.Fatalf("got %q, %v", got, err)
	}
}
