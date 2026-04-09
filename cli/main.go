package main

import (
	"context"

	"github.com/amp-labs/amp-common/logger"
	"github.com/eberle1080/repo-depot/cli/cmd"
)

func main() {
	ctx := context.Background()
	logger.ConfigureLogging(ctx, "rdcli")
	cmd.Execute()
}
