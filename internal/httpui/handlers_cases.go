package httpui

import (
	"net/http"
	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
)

func (s *Server) HandleListCases(w http.ResponseWriter, r *http.Request) {
	status := domain.Status(r.URL.Query().Get("status"))
	items, err := s.app.ListCasesByFindingTiming(status, r.URL.Query().Get("timeliness"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items)})
}

func (s *Server) HandleReviseCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReviseCaseCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.ReviseCase(r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleCreateCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateCaseCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.CreateCase(cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/cases/"+c.ID)
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) HandleGetCase(w http.ResponseWriter, r *http.Request) {
	detail, err := s.app.GetCase(r.PathValue("caseID"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) HandleAddLot(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddLotCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.AddLot(r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleAddLots(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddLotsCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.AddLots(r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
