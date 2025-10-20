package version

import (
	"fmt"
	"runtime"
)

var (
	Name      = "network-tunneler"
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
	Debug     = "true" // Set to "false" for release builds
	GoVersion = runtime.Version()
)

type Info struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	Debug     bool   `json:"debug"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func Get() Info {
	return Info{
		Name:      Name,
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		Debug:     IsDebug(),
		GoVersion: GoVersion,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// IsDebug returns true if this is a debug build
func IsDebug() bool {
	return Debug == "true"
}

func String() string {
	debugStr := ""
	if IsDebug() {
		debugStr = " [DEBUG]"
	}
	return fmt.Sprintf("%s version %s%s (commit: %s, built: %s, go: %s)",
		Name, Version, debugStr, Commit, BuildTime, GoVersion)
}

func Short() string {
	return fmt.Sprintf("%s %s", Name, Version)
}
