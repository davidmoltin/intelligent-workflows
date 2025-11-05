package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	apiURL     string
	apiToken   string
	outputJSON bool
)

var rootCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Intelligent Workflows CLI - Manage e-commerce workflows",
	Long: `The Intelligent Workflows CLI allows you to create, validate, deploy,
and manage e-commerce workflows from the command line.

Examples:
  workflow init my-approval-workflow --template approval
  workflow validate workflow.json
  workflow deploy workflow.json
  workflow list
  workflow test workflow.json --event event.json
  workflow logs <execution-id>`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize configuration
		initConfig()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.workflow-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "http://localhost:8080", "Workflow API URL")
	rootCmd.PersistentFlags().StringVar(&apiToken, "api-token", "", "API authentication token")
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "Output results in JSON format")

	// Bind flags to viper
	viper.BindPFlag("api.url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("api.token", rootCmd.PersistentFlags().Lookup("api-token"))
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".workflow-cli" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".workflow-cli")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("WORKFLOW")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if !outputJSON {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Override with flags if provided
	if apiURL != "" && apiURL != "http://localhost:8080" {
		viper.Set("api.url", apiURL)
	}
	if apiToken != "" {
		viper.Set("api.token", apiToken)
	}
}
