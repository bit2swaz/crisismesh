package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/bit2swaz/crisismesh/internal/config"
	"github.com/bit2swaz/crisismesh/internal/core"
	"github.com/bit2swaz/crisismesh/internal/discovery"
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
		// Use port in DB name to allow multiple local instances
		dbPath := fmt.Sprintf("crisis_%d.db", cfg.Port)
		db, err := store.Init(dbPath)
		if err != nil {
			slog.Error("Failed to init DB", "error", err)
			os.Exit(1)
		}

		// Get Node ID
		// Use port in identity file to allow multiple local instances
		identityPath := fmt.Sprintf("identity_%d.json", cfg.Port)
		nodeID, err := core.GenerateNodeID(identityPath)
		if err != nil {
			slog.Error("Failed to generate node ID", "error", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start Discovery Components
		peerChan := make(chan discovery.PeerInfo, 10)

		// 1. Heartbeat
		go func() {
			if err := discovery.StartHeartbeat(ctx, cfg.Port, nodeID, cfg.Nick); err != nil {
				slog.Error("Heartbeat failed", "error", err)
			}
		}()

		// 2. Listener
		go func() {
			if err := discovery.StartListener(ctx, cfg.Port, nodeID, peerChan); err != nil {
				slog.Error("Listener failed", "error", err)
			}
		}()

		// 3. Reaper
		go discovery.StartReaper(ctx, db)

		// 4. Peer Processor
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case info := <-peerChan:
					peer := store.Peer{
						ID:       info.ID,
						Nick:     info.Nick,
						Addr:     info.Addr,
						LastSeen: time.Now(),
						IsActive: true,
					}
					if err := store.UpsertPeer(db, peer); err != nil {
						slog.Error("Failed to upsert peer", "error", err)
					}
				}
			}
		}()

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
