package httpui

import (
	"net/http"
	"seed-vault-release/internal/application"
)

type Server struct {
	app *application.Service
	mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return securityHeaders(s.mux) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.HandleIndex)
	s.mux.HandleFunc("GET /assets/app.css", s.HandleCSS)
	s.mux.HandleFunc("GET /assets/workflow.css", s.HandleWorkflowCSS)
	s.mux.HandleFunc("GET /assets/app.js", s.HandleJS)
	s.mux.HandleFunc("GET /healthz", s.HandleHealth)
	s.mux.HandleFunc("GET /api/v1/cases", s.HandleListCases)
	s.mux.HandleFunc("POST /api/v1/cases", s.HandleCreateCase)
	s.mux.HandleFunc("GET /api/v1/cases/{caseID}", s.HandleGetCase)
	s.mux.HandleFunc("PATCH /api/v1/cases/{caseID}/base-data", s.HandleReviseCase)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/lots", s.HandleAddLot)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/lots/batch", s.HandleAddLots)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/sampling/generate", s.HandleGenerateSampling)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/sampling/confirm", s.HandleConfirmSampling)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/tests", s.HandleRecordTest)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/tests/batch", s.HandleRecordTests)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/findings/{findingID}/assign", s.HandleAssignFinding)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/findings/{findingID}/evidence", s.HandleAddFindingEvidence)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/findings/{findingID}/remediate", s.HandleRemediateFinding)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/findings/{findingID}/close", s.HandleCloseFinding)
	s.mux.HandleFunc("POST /api/v1/cases/{caseID}/approve", s.HandleApprove)
	s.mux.HandleFunc("GET /api/v1/certificates/{serial}/verify", s.HandleVerifyCertificate)
	s.mux.HandleFunc("GET /api/v1/certificates/verify", s.HandleVerifyCertificateQuery)
}
