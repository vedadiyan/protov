package options_test

import (
	"testing"

	"github.com/vedadiyan/protov/cmd/options"
)

func TestCompile_Run(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		wantErr bool
	}{
		struct {
			name    string
			wantErr bool
		}{
			name:    "basic",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			var c options.Compile
			c.Files = []string{`C:\Users\Pouya\Desktop\lab\users\service.proto`}
			c.Output = `C:\Users\Pouya\Desktop\lab\users\`
			gotErr := c.Run()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Run() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Run() succeeded unexpectedly")
			}
		})
	}
}
