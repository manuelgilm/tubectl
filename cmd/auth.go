/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"path/filepath"
	"tubectl/internal"
	"tubectl/internal/youtube"
	"github.com/spf13/cobra"
	"fmt"
)

var forceLogin bool

var authYoutubeCmd = &cobra.Command{
	Use: "youtube",
	Short: "Authenticate with Youtube via OAuth 2.0",
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
// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "A brief description of your command",
	Long: `....`,	

}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authYoutubeCmd)
	authYoutubeCmd.Flags().BoolVar(&forceLogin, "force", false, "Force re-authentication even if a valid token exists")
}
