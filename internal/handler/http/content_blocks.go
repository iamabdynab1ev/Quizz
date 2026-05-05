package http

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"

	"lms-arvand-backend/internal/domain"

	"github.com/go-chi/chi/v5"
)

type contentBlockUseCase interface {
	Create(ctx context.Context, params domain.CreateContentBlockParams) (domain.ContentBlock, error)
	GetByID(ctx context.Context, blockID string) (domain.ContentBlock, error)
	List(ctx context.Context, filter domain.ContentBlockListFilter) ([]domain.ContentBlock, int, error)
	Update(ctx context.Context, params domain.UpdateContentBlockParams) (domain.ContentBlock, error)
	Delete(ctx context.Context, blockID string) error
}

type ContentBlocksHandler struct {
	logger  *slog.Logger
	useCase contentBlockUseCase
}

func NewContentBlocksHandler(logger *slog.Logger, useCase contentBlockUseCase) *ContentBlocksHandler {
	return &ContentBlocksHandler{logger: logger, useCase: useCase}
}

func (h *ContentBlocksHandler) CreateContentBlock(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.CreateContentBlockParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	block, err := h.useCase.Create(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "create content block failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusCreated, block); err != nil {
		h.logger.ErrorContext(r.Context(), "create content block response failed", slog.String("error", err.Error()))
	}
}

func (h *ContentBlocksHandler) GetContentBlockByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	block, err := h.useCase.GetByID(r.Context(), chi.URLParam(r, "blockID"))
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "get content block failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, block); err != nil {
		h.logger.ErrorContext(r.Context(), "get content block response failed", slog.String("error", err.Error()))
	}
}

func (h *ContentBlocksHandler) ListContentBlocks(w nethttp.ResponseWriter, r *nethttp.Request) {
	filter, err := h.parseContentBlockListFilter(r)
	if err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	blocks, total, err := h.useCase.List(r.Context(), filter)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "list content blocks failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writePagedJSON(w, nethttp.StatusOK, blocks, total, 0, 0); err != nil {
		h.logger.ErrorContext(r.Context(), "list content blocks response failed", slog.String("error", err.Error()))
	}
}

func (h *ContentBlocksHandler) UpdateContentBlock(w nethttp.ResponseWriter, r *nethttp.Request) {
	var params domain.UpdateContentBlockParams
	if err := decodeJSON(w, r, &params, 1<<20); err != nil {
		writeDecodeError(w, err)
		return
	}

	params.ID = chi.URLParam(r, "blockID")

	block, err := h.useCase.Update(r.Context(), params)
	if err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "update content block failed", slog.String("error", err.Error()))
		}
		return
	}

	if err := writeJSON(w, nethttp.StatusOK, block); err != nil {
		h.logger.ErrorContext(r.Context(), "update content block response failed", slog.String("error", err.Error()))
	}
}

func (h *ContentBlocksHandler) DeleteContentBlock(w nethttp.ResponseWriter, r *nethttp.Request) {
	if err := h.useCase.Delete(r.Context(), chi.URLParam(r, "blockID")); err != nil {
		status := writeMappedError(w, err)
		if status >= nethttp.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "delete content block failed", slog.String("error", err.Error()))
		}
		return
	}

	w.WriteHeader(nethttp.StatusNoContent)
}

func (h *ContentBlocksHandler) parseContentBlockListFilter(r *nethttp.Request) (domain.ContentBlockListFilter, error) {
	query := r.URL.Query()
	filter := domain.ContentBlockListFilter{}

	if courseIDValue := query.Get("course_id"); courseIDValue != "" {
		filter.CourseID = &courseIDValue
	}

	if moduleIDValue := query.Get("module_id"); moduleIDValue != "" {
		filter.ModuleID = &moduleIDValue
	}

	if filter.CourseID != nil && filter.ModuleID != nil {
		return domain.ContentBlockListFilter{}, fmt.Errorf("only one of course_id or module_id is allowed")
	}

	return filter, nil
}
