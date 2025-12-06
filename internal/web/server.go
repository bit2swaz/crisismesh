package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
	"gorm.io/gorm"
)

//go:embed static/*
var staticFiles embed.FS

type Engine interface {
	GetNodeID() string
	PublishText(content string) error
}

type Server struct {
	db     *gorm.DB
	engine Engine
	port   int
}

func NewServer(db *gorm.DB, eng Engine, port int) *Server {
	return &Server{
		db:     db,
		engine: eng,
		port:   port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/map", s.handleMap)
	mux.HandleFunc("/api/messages", s.handleMessages)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/graph", s.handleGraph)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	slog.Info("Web server starting", "port", s.port)
	return srv.ListenAndServe()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(staticFiles, "static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handlePostMessage(w, r)
		return
	}

	var messages []store.Message
	if err := s.db.Order("timestamp desc").Limit(50).Find(&messages).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		for _, msg := range messages {
			colorClass := "text-green-500"
			if msg.SenderID == s.engine.GetNodeID() {
				colorClass = "text-cyan-400"
			}

			ts := time.Unix(msg.Timestamp, 0).Format("15:04:05")
			fmt.Fprintf(w, `<div class="mb-1 font-mono"><span class="text-gray-500">[%s]</span> <span class="font-bold %s">%s:</span> <span class="text-white">%s</span></div>`,
				ts, colorClass, msg.SenderID[:8], msg.Content)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	var content string

	if r.Header.Get("Content-Type") == "application/json" {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		content = req.Content
	} else {
		content = r.FormValue("content")
	}

	if content == "" {
		http.Error(w, "Content required", http.StatusBadRequest)
		return
	}

	if err := s.engine.PublishText(content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"node_id": s.engine.GetNodeID(),
		"peers":   0, // TODO: Expose peer count from engine
	}
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleMap(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(staticFiles, "static/map.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	var peers []store.Peer
	if err := s.db.Find(&peers).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type Node struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Color string `json:"color"`
		Shape string `json:"shape"`
	}
	type Link struct {
		From string `json:"from"`
		To   string `json:"to"`
	}

	nodes := []Node{}
	links := []Link{}

	myID := s.engine.GetNodeID()
	nodes = append(nodes, Node{
		ID:    myID,
		Label: "ME",
		Color: "#00FF00",
		Shape: "box",
	})

	for _, p := range peers {
		if p.ID == myID {
			continue
		}

		color := "#008800"
		if !p.IsActive {
			color = "#555555"
		}

		label := p.Nick
		if label == "" {
			label = p.ID[:8]
		}

		nodes = append(nodes, Node{
			ID:    p.ID,
			Label: label,
			Color: color,
			Shape: "dot",
		})

		links = append(links, Link{
			From: myID,
			To:   p.ID,
		})
	}

	resp := map[string]interface{}{
		"nodes": nodes,
		"links": links,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
