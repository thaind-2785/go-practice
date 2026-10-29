package main

import (
	"maps"
	"strings"
	"testing"
)

func TestParseUsers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{name: "single user", input: "admin:pw", want: map[string]string{"admin": "pw"}},
		{name: "several users with spaces", input: "admin:pw, bob:x:y", want: map[string]string{"admin": "pw", "bob": "x:y"}},
		{name: "empty", input: "", wantErr: true},
		{name: "missing colon", input: "admin", wantErr: true},
		{name: "empty password", input: "admin:", wantErr: true},
		{name: "empty user", input: ":pw", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUsers(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseUsers(%q) = %v, want an error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseUsers(%q): %v", tt.input, err)
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("parseUsers(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseUsers_ErrorDoesNotLeakPassword(t *testing.T) {
	_, err := parseUsers("admin:secret-value,broken")

	if err == nil || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("error = %v, want an error without the password", err)
	}
}
