package options

import "testing"

func TestParseSyntax(t *testing.T) {
	ParsePath([]byte(`
package options

func main() {

}
	
	`))
}

func TestModuleInit_Run(t *testing.T) {
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
			var x ModuleInit
			gotErr := x.Run()
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

func TestModuleBuild_Run(t *testing.T) {
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
			var x ModuleBuild
			gotErr := x.Run()
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
