package cmd

import (
	"context"
	"fmt"

	repodepotv1 "github.com/eberle1080/repo-depot/shared/gen/repodepot/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		client := repodepotv1.NewRepodepotServiceClient(conn)

		resp, err := client.CreateProject(context.Background(), &repodepotv1.CreateProjectRequest{
			Name: args[0],
		})
		if err != nil {
			return fmt.Errorf("create project: %w", err)
		}

		fmt.Printf("project %q created\n", resp.GetName())

		return nil
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects in the depot",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		client := repodepotv1.NewRepodepotServiceClient(conn)

		resp, err := client.ListProjects(context.Background(), &repodepotv1.ListProjectsRequest{})
		if err != nil {
			return fmt.Errorf("list projects: %w", err)
		}

		if len(resp.GetProjects()) == 0 {
			fmt.Println("no projects found")
			return nil
		}

		for _, p := range resp.GetProjects() {
			fmt.Println(p.GetName())
		}

		return nil
	},
}

var projectRenameCmd = &cobra.Command{
	Use:   "rename <new-name>",
	Short: "Rename the current project",
	Long: `Rename the current project. Discovered by walking up from the working directory
to find a git repo whose origin points at a depot project bare repo.
The origin remote is automatically updated to point at the renamed bare repo.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newName := args[0]

		oldName, repoRoot, err := currentProjectName()
		if err != nil {
			return fmt.Errorf("discover current project: %w", err)
		}

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		client := repodepotv1.NewRepodepotServiceClient(conn)

		resp, err := client.RenameProject(context.Background(), &repodepotv1.RenameProjectRequest{
			OldName: oldName,
			NewName: newName,
		})
		if err != nil {
			return fmt.Errorf("rename project: %w", err)
		}

		if err := gitRemoteSetURL(repoRoot, "origin", resp.GetNewBarePath()); err != nil {
			return fmt.Errorf("update origin remote: %w", err)
		}

		fmt.Printf("renamed %s → %s\n", oldName, newName)

		return nil
	},
}

var projectSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save and push a project (safe to delete workspace after)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		client := repodepotv1.NewRepodepotServiceClient(conn)

		_, err = client.SaveProject(context.Background(), &repodepotv1.SaveProjectRequest{
			Name: args[0],
		})
		if err != nil {
			return fmt.Errorf("save project: %w", err)
		}

		fmt.Println("project saved")

		return nil
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Archive and remove a project workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		client := repodepotv1.NewRepodepotServiceClient(conn)

		resp, err := client.DeleteProject(context.Background(), &repodepotv1.DeleteProjectRequest{
			Name: args[0],
		})
		if err != nil {
			return fmt.Errorf("delete project: %w", err)
		}

		fmt.Printf("archived to %s\n", resp.GetArchivePath())

		return nil
	},
}

var projectCheckoutCmd = &cobra.Command{
	Use:   "checkout <name>",
	Short: "Re-create a workspace for an existing project",
	Long: `Re-create the local workspace for a project whose bare repo already exists in the
depot. Use this after 'project delete' or on a fresh machine where the workspace is gone.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()

		_, err = repodepotv1.NewRepodepotServiceClient(conn).CheckoutProject(context.Background(), &repodepotv1.CheckoutProjectRequest{
			Name: args[0],
		})
		if err != nil {
			return fmt.Errorf("checkout project: %w", err)
		}

		fmt.Printf("project %q checked out\n", args[0])

		return nil
	},
}

func init() {
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectRenameCmd)
	projectCmd.AddCommand(projectSaveCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectCheckoutCmd)
	rootCmd.AddCommand(projectCmd)
}
