/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"path/filepath"
	"tubectl/internal"
	"tubectl/internal/prompt"
	"tubectl/internal/youtube"
	"github.com/spf13/cobra"
	"fmt"
)

var (
	forceLogin      bool
	mlflowUsername  string
	mlflowPassword  string
)

var authYoutubeCmd = &cobra.Command{
	Use: "youtube",
	Short: "Authenticate with Youtube via OAuth 2.0",
	Long: `Opens a browser for OAuth 2.0 consent. The obtained token is
saved to ~/.tubectl/auth/youtube.json for reuse.

Requires YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET environment
variables. Use --force to re-authenticate.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := TubeCtlHome()
		if err != nil {
			return err
		}

		tokenPath := filepath.Join(home, "auth", "youtube.json")
		// check if a token already exists
		if !forceLogin {
			if token, err := youtube.LoadToken(tokenPath); err == nil {
				if token.Valid(){
					fmt.Println("Already authenticated. Use `tubectl auth youtube --force` to re-authenticate.")
					return nil
				}
			}			
		}
		provider := youtube.NewYoutubeProvider(tokenPath)
		return provider.Login(cmd.Context(), internal.Options{})
	},
}
var authMlflowCmd = &cobra.Command{
	Use:   "mlflow",
	Short: "Authenticate with MLflow server (basic auth)",
	Long: `Saves MLflow credentials to ~/.tubectl/auth/mlflow.json for use
by other commands.

Credentials can be provided via --username/--password flags or by
setting the MLFLOW_USERNAME and MLFLOW_PASSWORD environment variables.
Use --force to overwrite existing credentials.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := TubeCtlHome()
		if err != nil {
			return err
		}
		credsPath := filepath.Join(home, "auth", "mlflow.json")

		if !forceLogin {
			if _, err := prompt.LoadCredentials(credsPath); err == nil {
				fmt.Println("Already authenticated. Use --force to re-authenticate.")
				return nil
			}
		}

		username := mlflowUsername
		password := mlflowPassword
		if username == "" {
			username = os.Getenv("MLFLOW_USERNAME")
		}
		if password == "" {
			password = os.Getenv("MLFLOW_PASSWORD")
		}
		if username == "" || password == "" {
			return fmt.Errorf("provide --username and --password or set MLFLOW_USERNAME/MLFLOW_PASSWORD environment variables")
		}

		provider := prompt.NewMLflowProvider(credsPath)
		return provider.Login(cmd.Context(), internal.Options{
			Username: username,
			Password: password,
		})
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication providers",

}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authYoutubeCmd)
	authYoutubeCmd.Flags().BoolVar(&forceLogin, "force", false, "Force re-authentication even if a valid token exists")
	authCmd.AddCommand(authMlflowCmd)
	authMlflowCmd.Flags().StringVar(&mlflowUsername, "username", "", "MLflow username")
	authMlflowCmd.Flags().StringVar(&mlflowPassword, "password", "", "MLflow password")
	authMlflowCmd.Flags().BoolVar(&forceLogin, "force", false, "Force re-authentication even if credentials exist")
}
