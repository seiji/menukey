package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolvedVersion(t *testing.T) {
	tests := []struct {
		name          string
		linkedVersion string
		info          *debug.BuildInfo
		want          string
	}{
		{
			name:          "linker version takes precedence",
			linkedVersion: "v1.2.3",
			info:          &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v9.9.9"}},
			want:          "v1.2.3",
		},
		{
			name:          "module version is used for go install",
			linkedVersion: developmentVersion,
			info:          &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "v1.2.3"}},
			want:          "v1.2.3",
		},
		{
			name:          "development build remains development version",
			linkedVersion: developmentVersion,
			info:          &debug.BuildInfo{Main: debug.Module{Path: modulePath, Version: "(devel)"}},
			want:          developmentVersion,
		},
		{
			name:          "other module is ignored",
			linkedVersion: developmentVersion,
			info:          &debug.BuildInfo{Main: debug.Module{Path: "example.com/other", Version: "v1.2.3"}},
			want:          developmentVersion,
		},
		{
			name:          "missing build information is ignored",
			linkedVersion: developmentVersion,
			want:          developmentVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvedVersion(tt.linkedVersion, tt.info); got != tt.want {
				t.Errorf("resolvedVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
