package cmd

import (
	"fmt"
	"os"

	"github.com/nov11/nacos-cli/internal/client"
	"github.com/nov11/nacos-cli/internal/terminal"
	"github.com/spf13/cobra"
)

var (
	serverAddr string
	namespace  string
	username   string
	password   string
)

var rootCmd = &cobra.Command{
	Use:   "nacos-cli",
	Short: "Nacos CLI - A command-line tool for managing Nacos configurations and skills",
	Long: `Nacos CLI is a powerful command-line tool for interacting with Nacos.
It supports configuration management, skill management, and provides an interactive terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior: start interactive terminal
		nacosClient := client.NewNacosClient(serverAddr, namespace, username, password)
		term := terminal.NewTerminal(nacosClient)
		if err := term.Start(); err != nil {
			checkError(err)
		}
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "server", "s", "127.0.0.1:8848", "Nacos server address")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace ID")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "nacos", "Username")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "nacos", "Password")
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
