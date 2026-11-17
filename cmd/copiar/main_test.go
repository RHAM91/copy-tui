package main

import "testing"

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantIcons bool
		wantErr   bool
	}{
		{name: "icons enabled by default", args: nil, wantIcons: true},
		{name: "no-icons flag disables icons", args: []string{"--no-icons"}, wantIcons: false},
		{name: "unknown flag is an error", args: []string{"--nope"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if opts.Icons != tt.wantIcons {
				t.Fatalf("Icons = %v, want %v", opts.Icons, tt.wantIcons)
			}
		})
	}
}
