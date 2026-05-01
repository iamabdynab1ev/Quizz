package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"strconv"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type certificateUseCase interface {
	Create(ctx context.Context, params domain.CreateCertificateParams) (domain.Certificate, error)
	GetByID(ctx context.Context, certificateID string) (domain.Certificate, error)
	GetByVerifyHash(ctx context.Context, verifyHash string) (domain.Certificate, error)
	List(ctx context.Context, filter domain.CertificateListFilter) ([]domain.Certificate, int, error)
}

type CertificatesHandler struct {
	logger  *slog.Logger
	useCase certificateUseCase
}

func NewCertificatesHandler(logger *slog.Logger, useCase certificateUseCase) *CertificatesHandler {
	return &CertificatesHandler{
		logger:  logger,
		useCase: useCase,
	}
}

func (h *CertificatesHandler) CreateCertificate(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateCertificateParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	certificate, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create certificate failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, certificate); err != nil {
		h.logger.ErrorContext(r.Context(), "create certificate response failed", slog.String("error", err.Error()))
	}
}

func (h *CertificatesHandler) GetCertificateByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	certificate, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "certificateID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get certificate failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := ensureOwnOrAdmin(r.Context(), &certificate.UserID); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "get certificate authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, certificate); err != nil {
		h.logger.ErrorContext(r.Context(), "get certificate response failed", slog.String("error", err.Error()))
	}
}

func (h *CertificatesHandler) VerifyCertificate(w nethttp.ResponseWriter, r *nethttp.Request) {
	certificate, err := h.useCase.GetByVerifyHash(r.Context(), chi.URLParam(r, "verifyHash"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "verify certificate failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, certificate); err != nil {
		h.logger.ErrorContext(r.Context(), "verify certificate response failed", slog.String("error", err.Error()))
	}
}

func (h *CertificatesHandler) ListCertificates(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseCertificateListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	filter.UserID, err = scopeUserID(r.Context(), filter.UserID)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError || status == nethttp.StatusForbidden || status == nethttp.StatusUnauthorized {
			h.logger.ErrorContext(r.Context(), "list certificates authorization failed", slog.String("error", err.Error()))
		}
		return
	}

	certificates, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list certificates failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, certificates, total, filter.Limit, filter.Offset); err != nil {
		h.logger.ErrorContext(r.Context(), "list certificates response failed", slog.String("error", err.Error()))
	}
}

func (h *CertificatesHandler) parseCertificateListFilter(r *nethttp.Request) (domain.CertificateListFilter, error) {
	query := r.URL.Query()

	filter := domain.CertificateListFilter{}

	if userIDValue := query.Get("user_id"); userIDValue != "" {
		filter.UserID = &userIDValue
	}

	if courseIDValue := query.Get("course_id"); courseIDValue != "" {
		filter.CourseID = &courseIDValue
	}

	if enrollmentIDValue := query.Get("enrollment_id"); enrollmentIDValue != "" {
		filter.EnrollmentID = &enrollmentIDValue
	}

	if limitValue := query.Get("limit"); limitValue != "" {
		parsed, err := strconv.Atoi(limitValue)
		if err != nil {
			return domain.CertificateListFilter{}, fmt.Errorf("limit must be integer")
		}
		filter.Limit = parsed
	}

	if offsetValue := query.Get("offset"); offsetValue != "" {
		parsed, err := strconv.Atoi(offsetValue)
		if err != nil {
			return domain.CertificateListFilter{}, fmt.Errorf("offset must be integer")
		}
		filter.Offset = parsed
	}

	return filter, nil
}
