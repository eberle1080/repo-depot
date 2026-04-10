#!/bin/bash

# Make bash more strict
set -euo pipefail

# It's good practice to not assume the running CWD, this helps with that
script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)

# cd to where this script lives
cd "${script_dir}"

# Find the root of the project and cd there
cd "$(git rev-parse --show-toplevel)"

# Gather some basic info about the git repo and this build
commit=$(git rev-parse HEAD)
branch=${BRANCH_NAME:-$(git rev-parse --abbrev-ref HEAD)}
git_date=$(git show --no-patch --no-notes --pretty='%cI' "${commit}")
build_time=$(date +%Y-%m-%dT%H:%M:%S%z)
build_host=$(hostname -f)
build_user=$(id -un)
go_vers=$(go version)

# Get amp-common version from go.mod
amp_common_version=$(grep 'github.com/amp-labs/amp-common' server/go.mod | awk '{print $2}')

# Create a JSON file with the build info. This will be embedded in the
# binary via go's go:embed capability.
cat << EOL > shared/build/info.json
{
    "git_commit": "${commit}",
    "git_branch": "${branch}",
    "git_date": "${git_date}",
    "build_time": "${build_time}",
    "build_host": "${build_host}",
    "build_user": "${build_user}",
    "go_version": "${go_vers}",
    "dependencies": {
        "amp-common": "${amp_common_version}"
    }
}
EOL
