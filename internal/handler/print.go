package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/printdreams/cryptocash-ton-battery/internal/abuse"
	"github.com/printdreams/cryptocash-ton-battery/internal/auth"
	"github.com/printdreams/cryptocash-ton-battery/internal/ledger"
	"github.com/printdreams/cryptocash-ton-battery/internal/policy"
	"github.com/printdreams/cryptocash-ton-battery/internal/price"
	"github.com/printdreams/cryptocash-ton-battery/internal/print"
	"github.com/printdreams/cryptocash-ton-battery/internal/relayer"
)

type PrintHandler struct {
	Store      *print.Store
	Ledger     *ledger.Store
	Sender     *relayer.Sender
	Price      *price.Oracle
	PolCfg     policy.Config
	Cfg        print.Config
	Limiter    *abuse.Limiter
	Killswitch *abuse.Killswitch
}

func NewPrintHandler(st *print.Store, l *ledger.Store, s *relayer.Sender, pr *price.Oracle, pol policy.Config, cfg print.Config, lim *abuse.Limiter, ks *abuse.Killswitch) *PrintHandler {
	return &PrintHandler{Store: st, Ledger: l, Sender: s, Price: pr, PolCfg: pol, Cfg: cfg, Limiter: lim, Killswitch: ks}
}

func (h *PrintHandler) nanoPerCharge(ctx context.Context) int64 {
	cfg := h.PolCfg
	if h.Price != nil {
		if n := h.Price.NanoPerCharge(ctx); n > 0 {
			cfg.NanoPerCharge = n
		}
	}
	return cfg.NanoPerCharge
}

type printQuoteRequest struct {
	ToAddress string `json:"toAddress"`
}

// Quote godoc
// @Summary      Quote a printed (gift) wallet
// @Description  Computes the buffer and charge cost for printing a new wallet and stores the quote
// @Tags         print
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      printQuoteRequest  true  "destination address"
// @Success      200   {object}  handler.PrintQuoteResponse
// @Failure      400   {object}  handler.ErrorResponse
// @Failure      401   {object}  handler.ErrorResponse
// @Router       /print/quote [post]
func (h *PrintHandler) Quote(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req printQuoteRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}
	if req.ToAddress == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing-to-address"})
		return
	}

	bufferNano := print.BufferNano(h.Cfg)
	polCfg := h.PolCfg
	polCfg.NanoPerCharge = h.nanoPerCharge(r.Context())
	charge := policy.ComputeCharge(bufferNano, polCfg)

	now := time.Now().UTC()
	job := &print.Job{
		UserID:     userID,
		ToAddress:  req.ToAddress,
		BufferNano: bufferNano,
		Charge:     charge,
		Status:     "quoted",
		CreatedAt:  now,
		ExpiresAt:  now.Add(h.Cfg.QuoteTTL),
	}
	id, err := h.Store.Create(r.Context(), job)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store-failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"quoteId":   id,
		"bufferTON": print.NanoToTON(bufferNano),
		"charge":    charge,
		"expiresAt": job.ExpiresAt.Unix(),
	})
}

type printExecuteRequest struct {
	QuoteID string `json:"quoteId"`
}

// Execute godoc
// @Summary      Execute a printed wallet
// @Description  Reserves charges, sends the buffer to the new wallet (buffer-first), settles or releases on result
// @Tags         print
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      printExecuteRequest  true  "quote id"
// @Success      200   {object}  handler.PrintExecuteResponse
// @Failure      400   {object}  handler.ErrorResponse
// @Failure      401   {object}  handler.ErrorResponse
// @Failure      502   {object}  handler.ErrorResponse
// @Router       /print/execute [post]
func (h *PrintHandler) Execute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req printExecuteRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed-request"})
		return
	}
	if req.QuoteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing-quote-id"})
		return
	}

	if reason := abuseReject(r, userID, h.Killswitch, h.Limiter); reason != "" {
		rejectAbuse(w, reason)
		return
	}

	job, err := h.Store.Get(r.Context(), req.QuoteID)
	if err != nil {
		if errors.Is(err, print.ErrNotFound) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "print-job-not-found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "store-failed"})
		return
	}
	if job.UserID != userID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if job.Status != "quoted" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "already-processed", "status": job.Status})
		return
	}
	if time.Now().UTC().After(job.ExpiresAt) {
		_ = h.Store.Update(r.Context(), job.ID, map[string]interface{}{"status": "expired"})
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "quote-expired"})
		return
	}

	if h.Sender == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "treasury-not-ready"})
		return
	}

	if job.Charge > 0 {
		if _, rerr := h.Ledger.Reserve(r.Context(), userID, job.Charge, ledger.Ref{
			IdempotencyKey: "print-reserve:" + job.ID,
			Type:           "print",
		}); rerr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "insufficient-charges"})
			return
		}
	}

	_ = h.Store.Update(r.Context(), job.ID, map[string]interface{}{"status": "buffer_pending"})

	sctx, scancel := context.WithTimeout(r.Context(), 90*time.Second)
	txHash, serr := h.Sender.Send(sctx, job.ToAddress, print.NanoToTON(job.BufferNano), "battery-print-buffer")
	scancel()

	if serr != nil {
		if job.Charge > 0 {
			_, _ = h.Ledger.Release(r.Context(), userID, job.Charge, ledger.Ref{
				IdempotencyKey: "print-release:" + job.ID,
				Type:           "print",
			})
		}
		_ = h.Store.Update(r.Context(), job.ID, map[string]interface{}{"status": "failed"})
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "buffer-send-failed", "detail": serr.Error()})
		return
	}

	if job.Charge > 0 {
		_, _ = h.Ledger.Settle(r.Context(), userID, job.Charge, ledger.Ref{
			IdempotencyKey: "print-settle:" + job.ID,
			Type:           "print",
		})
	}
	_ = h.Store.Update(r.Context(), job.ID, map[string]interface{}{
		"status":  "completed",
		"tx_hash": txHash,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"quoteId": job.ID,
		"status":  "completed",
		"txHash":  txHash,
		"charge":  job.Charge,
	})
}

// Status godoc
// @Summary      Print job status
// @Description  Returns the status of a print job
// @Tags         print
// @Produce      json
// @Security     BearerAuth
// @Param        id  path      string  true  "print job id"
// @Success      200  {object}  handler.PrintStatusResponse
// @Failure      401  {object}  handler.ErrorResponse
// @Failure      404  {object}  handler.ErrorResponse
// @Router       /print/{id} [get]
func (h *PrintHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	id := chi.URLParam(r, "id")
	job, err := h.Store.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "print-job-not-found"})
		return
	}
	if job.UserID != userID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"quoteId":   job.ID,
		"status":    job.Status,
		"toAddress": job.ToAddress,
		"bufferTON": print.NanoToTON(job.BufferNano),
		"charge":    job.Charge,
		"txHash":    job.TxHash,
	})
}
