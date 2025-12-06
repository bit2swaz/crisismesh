package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bit2swaz/crisismesh/internal/config"
	"github.com/bit2swaz/crisismesh/internal/core"
	"github.com/bit2swaz/crisismesh/internal/engine"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
	"github.com/bit2swaz/crisismesh/internal/tui"
	"github.com/bit2swaz/crisismesh/internal/web"
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
		dbPath := fmt.Sprintf("crisis_%d.db", cfg.Port)
		db, err := store.Init(dbPath)
		if err != nil {
			slog.Error("Failed to init DB", "error", err)
			os.Exit(1)
		}
		identityPath := fmt.Sprintf("identity_%d.json", cfg.Port)
		id, err := core.LoadOrGenerateIdentity(identityPath)
		if err != nil {
			slog.Error("Failed to load identity", "error", err)
			os.Exit(1)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		tm := transport.NewManager()
		eng := engine.NewGossipEngine(db, tm, id.NodeID, cfg.Nick, cfg.Port, id.PubKey, id.PrivKey)
		if err := eng.Start(ctx); err != nil {
			slog.Error("Failed to start gossip engine", "error", err)
			os.Exit(1)
		}

		// Start Web Server
		webSrv := web.NewServer(db, eng, cfg.WebPort)
		go func() {
			if err := webSrv.Start(ctx); err != nil {
				slog.Error("Web server failed", "error", err)
				os.Exit(1) // Force exit if web server fails
			}
		}()

		// Check if we are in a headless environment (no TTY)
		// If so, don't start TUI, just block
		if os.Getenv("CRISIS_HEADLESS") == "true" {
			slog.Info("Running in HEADLESS mode (No TUI)")
			// Keep main thread alive
			select {
			case <-ctx.Done():
			}
		} else {
			if err := tui.StartTUI(db, id.NodeID, eng.MsgUpdates, eng.PeerUpdates, eng); err != nil {
				slog.Error("TUI failed", "error", err)
				// Don't exit on TUI failure in some cases, but here we probably should
				// os.Exit(1)
			}
		}
	},
}

func init() {
	startCmd.Flags().IntVar(&cfg.Port, "port", 9000, "Port to listen on")
	startCmd.Flags().IntVar(&cfg.WebPort, "web-port", 8080, "Port for web interface")
	startCmd.Flags().StringVar(&cfg.Nick, "nick", "", "Nickname (required)")
	startCmd.Flags().StringVar(&cfg.Room, "room", "lobby", "Room to join")
	startCmd.MarkFlagRequired("nick")
	rootCmd.AddCommand(startCmd)
}
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
