package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/bit2swaz/crisismesh/internal/config"
	"github.com/bit2swaz/crisismesh/internal/core"
	"github.com/bit2swaz/crisismesh/internal/engine"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/bit2swaz/crisismesh/internal/transport"
	"github.com/bit2swaz/crisismesh/internal/tui"
	"github.com/bit2swaz/crisismesh/internal/uplink"
	"github.com/bit2swaz/crisismesh/internal/utils"
	"github.com/bit2swaz/crisismesh/internal/web"
	"github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
)

var cfg config.Config
var discordWebhook string

var rootCmd = &cobra.Command{
	Use:   "crisis",
	Short: "CrisisMesh CLI tool",
}
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the CrisisMesh service",
	Run: func(cmd *cobra.Command, args []string) {
		// Smart Port Logic: If user changed Gossip Port but left Web Port at default (8080),
		// auto-adjust Web Port to avoid conflicts.
		if cfg.Port != 9000 && cfg.WebPort == 8080 {
			offset := cfg.Port - 9000
			cfg.WebPort = 8080 + offset
			fmt.Printf("Auto-adjusting Web Port to %d (to match Gossip Port offset)\n", cfg.WebPort)
		}

		// Validate ports are available
		if err := checkPort(cfg.Port); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Gossip port %d is already in use.\n", cfg.Port)
			os.Exit(1)
		}
		if err := checkPort(cfg.WebPort); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Web port %d is already in use.\n", cfg.WebPort)
			os.Exit(1)
		}

		slog.Info("Starting CrisisMesh", "port", cfg.Port, "nick", cfg.Nick)
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

		// Uplink Service Integration
		if discordWebhook != "" {
			slog.Info("Initializing Uplink Service", "webhook", "REDACTED")
			upService := uplink.NewService(discordWebhook)
			eng.UplinkChan = make(chan store.Message, 100)
			upService.Start(eng.UplinkChan)
		}

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

		// Generate QR Code
		ip, _ := utils.GetOutboundIP()
		url := fmt.Sprintf("http://%s:%d", ip, cfg.WebPort)
		qr, _ := qrcode.New(url, qrcode.Medium)
		qrAscii := qr.ToString(false)

		// Print QR to stdout for logs
		fmt.Println("\nSCAN TO JOIN MESH:")
		fmt.Println(qrAscii)
		fmt.Println("URL:", url)

		// Check if we are in a headless environment (no TTY)
		// If so, don't start TUI, just block
		if os.Getenv("CRISIS_HEADLESS") == "true" {
			slog.Info("Running in HEADLESS mode (No TUI)")
			// Keep main thread alive
			select {
			case <-ctx.Done():
			}
		} else {
			if err := tui.StartTUI(db, id.NodeID, eng.MsgUpdates, eng.PeerUpdates, eng, qrAscii); err != nil {
				slog.Error("TUI failed", "error", err)
				// Don't exit on TUI failure in some cases, but here we probably should
				// os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().IntVarP(&cfg.Port, "port", "p", 9000, "Port to listen on")
	startCmd.Flags().IntVarP(&cfg.WebPort, "web-port", "w", 8080, "Web interface port")
	startCmd.Flags().StringVarP(&cfg.Nick, "nick", "n", "Anonymous", "Nickname")
	startCmd.Flags().StringVar(&discordWebhook, "discord-webhook", "", "Discord Webhook URL for Uplink Service")
}
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkPort(port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}
