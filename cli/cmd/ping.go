package cmd

import (
	"context"
	"fmt"

	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var pingCmd = &cobra.Command{
	Use:   "ping [message]",
	Short: "Ping the server",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		msg := "ping"
		if len(args) > 0 {
			msg = args[0]
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		client := repodepotv1.NewRepodepotServiceClient(conn)

		resp, err := client.Ping(context.Background(), &repodepotv1.PingRequest{Message: msg})
		if err != nil {
			return fmt.Errorf("ping: %w", err)
		}

		fmt.Printf("response: %s\n", resp.GetMessage())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
