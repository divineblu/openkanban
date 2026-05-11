package update

import "testing"

func TestCheckResultUpdateHint(t *testing.T) {
	tests := []struct {
		name   string
		result CheckResult
		want   string
	}{
		{
			name: "homebrew formula",
			result: CheckResult{
				InstallMethod: InstallHomebrew,
			},
			want: "brew upgrade openkanban",
		},
		{
			name: "homebrew cask",
			result: CheckResult{
				InstallMethod: InstallHomebrewCask,
			},
			want: "brew upgrade --cask openkanban",
		},
		{
			name: "go install",
			result: CheckResult{
				InstallMethod: InstallGo,
			},
			want: "go install github.com/divineblu/openkanban@latest",
		},
		{
			name: "unknown",
			result: CheckResult{
				InstallMethod: InstallUnknown,
				ReleaseURL:    "https://github.com/divineblu/openkanban/releases/tag/v1.2.3",
			},
			want: "https://github.com/divineblu/openkanban/releases/tag/v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.UpdateHint(); got != tt.want {
				t.Fatalf("UpdateHint() = %q, want %q", got, tt.want)
			}
		})
	}
}
