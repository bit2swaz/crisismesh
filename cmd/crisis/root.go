package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/bit2swaz/crisismesh/internal/config"
	"github.com/bit2swaz/crisismesh/internal/core"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/tui"
	"github.com/spf13/cobra"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "crisis",
	Short: "CrisisMesh CLI tool",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the CrisisMesh service",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Starting CrisisMesh", "port", cfg.Port, "nick", cfg.Nick, "room", cfg.Room)

		// Initialize DB
		db, err := store.Init("crisis.db")
		if err != nil {
			slog.Error("Failed to init DB", "error", err)
			os.Exit(1)
		}

		// Get Node ID
		nodeID, err := core.GenerateNodeID()
		if err != nil {
			slog.Error("Failed to generate node ID", "error", err)
			os.Exit(1)
		}

		// Start TUI
		if err := tui.StartTUI(db, nodeID); err != nil {
			slog.Error("TUI failed", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	startCmd.Flags().IntVar(&cfg.Port, "port", 9000, "Port to listen on")
	startCmd.Flags().StringVar(&cfg.Nick, "nick", "", "Nickname (required)")
	startCmd.Flags().StringVar(&cfg.Room, "room", "lobby", "Room to join")

	// DBPath isn't in the flags requirement, but it's in the config struct.
	// Leaving it unflagged for now as per instructions.

	startCmd.MarkFlagRequired("nick")

	rootCmd.AddCommand(startCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
