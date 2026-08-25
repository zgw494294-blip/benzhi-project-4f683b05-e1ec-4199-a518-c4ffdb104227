package httpui

import (
	"net/http"
	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
	"strconv"
)

func (s *Server) HandleVerifyCertificate(w http.ResponseWriter, r *http.Request) {
	serial, err := strconv.ParseUint(r.PathValue("serial"), 10, 64)
	if err != nil || serial == 0 {
		writeError(w, r, domain.Invalid("凭据序号无效"))
		return
	}
	result, err := s.app.VerifyCertificate(serial)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) HandleVerifyCertificateQuery(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	serialRaw := values.Get("serialNumber")
	if values.Get("serial") != "" {
		if serialRaw != "" {
			writeError(w, r, domain.Invalid("凭据序号条件重复"))
			return
		}
		serialRaw = values.Get("serial")
	}
	accessionCode := values.Get("accessionCode")
	if values.Get("caseCode") != "" {
		if accessionCode != "" {
			writeError(w, r, domain.Invalid("案卷编号条件重复"))
			return
		}
		accessionCode = values.Get("caseCode")
	}
	manifestDigest := values.Get("manifestDigest")
	if values.Get("digest") != "" {
		if manifestDigest != "" {
			writeError(w, r, domain.Invalid("清单摘要条件重复"))
			return
		}
		manifestDigest = values.Get("digest")
	}
	query := application.CertificateQuery{AccessionCode: accessionCode, ManifestDigest: manifestDigest}
	if raw := serialRaw; raw != "" {
		serial, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || serial == 0 {
			writeError(w, r, domain.Invalid("凭据序号无效"))
			return
		}
		query.SerialNumber = serial
	}
	result, err := s.app.VerifyCertificateQuery(query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
