package httpui

import (
	"net/http"
	"seed-vault-release/internal/application"
)

func (s *Server) HandleGenerateSampling(w http.ResponseWriter, r *http.Request) {
	var ctx application.Context
	if err := decodeJSON(w, r, &ctx); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &ctx); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.GenerateSampling(r.PathValue("caseID"), ctx)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleConfirmSampling(w http.ResponseWriter, r *http.Request) {
	var cmd application.ConfirmSamplingCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.ConfirmSampling(r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleRecordTest(w http.ResponseWriter, r *http.Request) {
	var cmd application.RecordTestCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.RecordTest(r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleRecordTests(w http.ResponseWriter, r *http.Request) {
	var cmd application.RecordTestsCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.RecordTests(r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleAssignFinding(w http.ResponseWriter, r *http.Request) {
	var cmd application.AssignFindingCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.AssignFinding(r.PathValue("caseID"), r.PathValue("findingID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleAddFindingEvidence(w http.ResponseWriter, r *http.Request) {
	var cmd application.AddEvidenceCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.AddFindingEvidence(r.PathValue("caseID"), r.PathValue("findingID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleRemediateFinding(w http.ResponseWriter, r *http.Request) {
	var cmd application.RemediateCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.RemediateFinding(r.PathValue("caseID"), r.PathValue("findingID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleCloseFinding(w http.ResponseWriter, r *http.Request) {
	var cmd application.CloseFindingCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.CloseFinding(r.PathValue("caseID"), r.PathValue("findingID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) HandleApprove(w http.ResponseWriter, r *http.Request) {
	var cmd application.ApproveCommand
	if err := decodeJSON(w, r, &cmd); err != nil {
		writeError(w, r, err)
		return
	}
	if err := enrichContext(r, &cmd.Context); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := s.app.Approve(r.PathValue("caseID"), cmd)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
