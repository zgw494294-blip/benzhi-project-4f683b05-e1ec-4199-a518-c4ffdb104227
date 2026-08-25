package httpui

import "net/http"

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	b, err := assets.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "资源不可用", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) HandleCSS(w http.ResponseWriter, r *http.Request) {
	b, err := assets.ReadFile("web/app.css")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) HandleWorkflowCSS(w http.ResponseWriter, r *http.Request) {
	b, err := assets.ReadFile("web/workflow.css")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) HandleJS(w http.ResponseWriter, r *http.Request) {
	b, err := assets.ReadFile("web/app.js")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "seed-vault-release"})
}
