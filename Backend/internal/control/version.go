package control

import (
	"os/exec"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var (
	buildVersion = "dev"
	buildCommit  = ""
	buildTime    = ""

	versionInfoOnce sync.Once
	versionInfoData map[string]any
)

func resolveVersionInfo() map[string]any {
	versionInfoOnce.Do(func() {
		version := strings.TrimSpace(buildVersion)
		commit := strings.TrimSpace(buildCommit)
		builtAt := strings.TrimSpace(buildTime)
		modified := false

		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if commit == "" {
						commit = strings.TrimSpace(setting.Value)
					}
				case "vcs.time":
					if builtAt == "" {
						builtAt = strings.TrimSpace(setting.Value)
					}
				case "vcs.modified":
					modified = strings.EqualFold(strings.TrimSpace(setting.Value), "true")
				}
			}
		}

		if commit == "" {
			commit = gitOutput("rev-parse", "HEAD")
		}
		if commit == "" {
			commit = "unknown"
		}
		shortCommit := commit
		if len(shortCommit) > 8 {
			shortCommit = shortCommit[:8]
		}

		if builtAt == "" {
			builtAt = gitOutput("show", "-s", "--format=%cI", "HEAD")
		}
		if builtAt == "" {
			builtAt = time.Now().UTC().Format(time.RFC3339)
		}

		if version == "" || version == "dev" {
			tag := gitOutput("describe", "--tags", "--abbrev=0")
			if tag != "" {
				version = tag
			} else {
				version = "dev"
			}
		}
		if version == "dev" || strings.Contains(version, "SNAPSHOT") {
			version = version + "+" + shortCommit
		}
		if modified {
			version = version + "-dirty"
		}

		versionInfoData = map[string]any{
			"version":      version,
			"commit":       commit,
			"short_commit": shortCommit,
			"build_time":   builtAt,
			"modified":     modified,
		}
	})

	return versionInfoData
}

func gitOutput(args ...string) string {
	cmd := exec.Command("git", args...)
	raw, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
