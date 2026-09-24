package weclawbot

import (
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

const modulePath = "github.com/go-sdk/weclawbot"

var versionNumbers = regexp.MustCompile(`\d+`)

func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return "0.0.0"
	}
	if info.Main.Path == modulePath {
		if version := normalizeVersion(info.Main.Version); version != "" {
			return version
		}
	}
	for _, dependency := range info.Deps {
		if dependency.Path == modulePath {
			if version := normalizeVersion(dependency.Version); version != "" {
				return version
			}
		}
	}
	return "0.0.0"
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if version == "" || version == "(devel)" {
		return ""
	}
	return version
}

func clientVersion(version string) string {
	parts := versionNumbers.FindAllString(normalizeVersion(version), 3)
	values := [3]uint64{}
	for i, part := range parts {
		value, _ := strconv.ParseUint(part, 10, 64)
		values[i] = value & 0xff
	}
	encoded := values[0]<<16 | values[1]<<8 | values[2]
	return strconv.FormatUint(encoded, 10)
}

func defaultBotAgent(version string) string {
	return "weclawbot/" + normalizeVersion(version)
}

func sanitizeBotAgent(agent, fallback string) string {
	agent = strings.TrimSpace(agent)
	if agent == "" {
		return fallback
	}
	var builder strings.Builder
	builder.Grow(len(agent))
	for _, r := range agent {
		if r < 0x20 || r > 0x7e {
			continue
		}
		builder.WriteRune(r)
		if builder.Len() >= 256 {
			break
		}
	}
	value := strings.TrimSpace(builder.String())
	if value == "" {
		return fallback
	}
	return value
}
