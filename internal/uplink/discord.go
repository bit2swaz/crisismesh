package uplink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
)

type Service struct {
	WebhookURL string
	client     *http.Client
}

func NewService(url string) *Service {
	return &Service{
		WebhookURL: url,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) Start(msgChan <-chan store.Message) {
	go func() {
		for msg := range msgChan {
			// Filter: Only SOS (Priority 2) or explicit /uplink command
			isSOS := msg.Priority == 2
			isUplinkCmd := strings.HasPrefix(msg.Content, "/uplink")

			if !isSOS && !isUplinkCmd {
				continue
			}

			// Format Content
			content := msg.Content
			if isUplinkCmd {
				content = strings.TrimPrefix(content, "/uplink")
				content = strings.TrimSpace(content)
			}

			// Format Location
			locationStr := "Unknown"
			if msg.Lat != 0 && msg.Long != 0 {
				locationStr = fmt.Sprintf("%.4f, %.4f\n[Open in Maps](https://maps.google.com/?q=%f,%f)", msg.Lat, msg.Long, msg.Lat, msg.Long)
			}

			// Construct Discord Payload
			discordMsg := fmt.Sprintf("📡 **[MESH RELAY]**\n**User:** %s\n**Message:** %s\n**Location:** %s",
				msg.Author,
				content,
				locationStr,
			)

			payload := map[string]string{
				"content": discordMsg,
			}

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				slog.Error("Failed to marshal uplink payload", "error", err)
				continue
			}

			// Send Request
			resp, err := s.client.Post(s.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				slog.Error("Failed to send uplink request", "error", err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				// Print to stdout for TUI pickup
				fmt.Println("[UPLINK] Relayed to Cloud")
			} else {
				slog.Error("Uplink returned non-200 status", "status", resp.Status)
			}
		}
	}()
}
