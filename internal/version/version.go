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
	GoVersion = runtime.Version()
)

type Info struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
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
		GoVersion: GoVersion,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

func String() string {
	return fmt.Sprintf("%s version %s (commit: %s, built: %s, go: %s)",
		Name, Version, Commit, BuildTime, GoVersion)
}

func Short() string {
	return fmt.Sprintf("%s %s", Name, Version)
}
