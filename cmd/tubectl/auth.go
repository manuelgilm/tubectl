package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"tubectl/internal/youtube"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with YouTube via OAuth2",
	Long: `Opens a Google OAuth2 consent URL. Open it in your browser,
grant access, then paste the authorization code back here.
The token is saved to ~/.config/tubectl/token.json.`,
	RunE: runAuthLogin,
}

var forceLogin bool

func init() {
	authCmd.AddCommand(authLoginCmd)
	authLoginCmd.Flags().BoolVar(&forceLogin, "force", false, "Force re-authentication even if a valid token exists")
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	cfg, err := oauthCfg()
	if err != nil {
		return err
	}

	path := tokenFilePath()

	// Check if a token already exists.
	if !forceLogin {
		if tok, err := youtube.LoadToken(path); err == nil {
			if tok.Valid() {
				fmt.Println("Already authenticated. Use `tubectl auth login --force` to re-authenticate.")
				return nil
			}
			// Token exists but is expired — try a silent refresh.
			if tok.RefreshToken != "" {
				fmt.Println("Access token expired, refreshing...")
				refreshed, err := cfg.Refresh(cmd.Context(), tok.RefreshToken)
				if err == nil {
					if err := youtube.SaveToken(path, refreshed); err != nil {
						return fmt.Errorf("saving refreshed token: %w", err)
					}
					fmt.Println("Token refreshed successfully.")
					return nil
				}
				fmt.Println("Refresh failed, starting new login flow...")
			}
		}
	}

	fmt.Println("Open the following URL in your browser:")
	fmt.Println()
	fmt.Println(" ", cfg.AuthCodeURL("tubectl"))
	fmt.Println()
	fmt.Print("Paste the authorization code here: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	code := scanner.Text()
	if code == "" {
		return fmt.Errorf("no code provided")
	}

	tok, err := cfg.Exchange(cmd.Context(), code)
	if err != nil {
		return fmt.Errorf("exchanging code: %w", err)
	}

	if err := youtube.SaveToken(path, tok); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	fmt.Printf("Authenticated successfully. Token saved to %s\n", path)
	return nil
}
