// Package build provides build-time information such as version, commit hash, and build timestamp.
package build //nolint:revive // Package name intentionally uses domain-specific name

import (
	_ "embed"

	"github.com/amp-labs/amp-common/build"
)

//go:generate ../../scripts/gen_version.sh
//go:embed info.json
var buildInfo string

// Get returns the embedded build information as a JSON string.
// Returns empty JSON object if build info is not available.
func Get() string {
	if len(buildInfo) == 0 {
		return "{}"
	}

	return buildInfo
}

// ReadInfo parses the build information into a structured format.
// Returns the parsed build info and a boolean indicating success.
func ReadInfo() (*build.Info, bool) {
	return build.Parse(Get())
}
