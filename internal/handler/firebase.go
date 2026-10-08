package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/api/iterator"

	"github.com/printdreams/cryptocash-ton-battery/internal/firebase"
)

type FirebaseHandler struct {
	FB *firebase.Clients
}

func NewFirebaseHandler(fb *firebase.Clients) *FirebaseHandler {
	return &FirebaseHandler{FB: fb}
}

// Status godoc
// @Summary      Firebase connection status
// @Description  Verifies the Firestore connection with a live round-trip and reports whether Firebase is reachable
// @Tags         firebase
// @Produce      json
// @Success      200  {object}  handler.FirebaseStatusResponse
// @Failure      503  {object}  handler.FirebaseStatusResponse
// @Router       /firebase/status [get]
func (h *FirebaseHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.FB == nil || h.FB.App == nil || h.FB.Firestore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"connected": false,
			"message":   "Firebase is not initialized",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.FB.Firestore.Collections(ctx).Next(); err != nil && err != iterator.Done {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"connected": false,
			"message":   "Firestore connection failed",
			"error":     err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"connected": true,
		"message":   "Firestore connection OK",
	})
}
