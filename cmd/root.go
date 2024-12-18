package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yckao/gta/pkg/logger"
)

var (
	cfgFile   string
	project   string
	user      string
	ttl       time.Duration
	verbosity string
	logFormat string
	quietMode bool
	dryRun    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gta",
	Short: "Grant Temporary Access - Manage temporary IAM roles across cloud providers",
	Long: `Grant Temporary Access (gta) is a CLI tool for managing temporary IAM roles
across different cloud providers. It currently supports GCP and allows you to
grant temporary permissions that are automatically revoked when the program exits.`,
	PersistentPreRunE: setupLogging,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("please specify a command (e.g., grant, list)")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	flags := rootCmd.PersistentFlags()
	flags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gta.yaml)")
	flags.StringVarP(&verbosity, "verbosity", "v", "info", "log level (debug, info, warn, error)")
	flags.StringVar(&logFormat, "format", "plain", "log format (plain, json)")
	flags.BoolVarP(&quietMode, "quiet", "q", false, "quiet mode, only show errors")

	// Add commands
	rootCmd.AddCommand(grantCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(cleanCmd)
}

// setupLogging configures the logging system based on command-line flags
func setupLogging(cmd *cobra.Command, args []string) error {
	// Set up logging based on verbosity flags
	if quietMode {
		logger.SetLevel(logger.LevelError)
	} else {
		level, err := logger.ParseLevel(verbosity)
		if err != nil {
			return err
		}
		logger.SetLevel(level)
	}

	// Set up logging format
	format, err := logger.ParseFormat(logFormat)
	if err != nil {
		return err
	}
	if err := logger.SetFormat(format); err != nil {
		return err
	}

	logger.Debug("Starting command execution: %s", cmd.Name())
	logger.Debug("Arguments: %v", args)
	return nil
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Error("Error finding home directory: %v", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gta" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gta")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logger.Debug("Using config file: %s", viper.ConfigFileUsed())
	}
}
