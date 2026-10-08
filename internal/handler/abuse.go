package handler

import (
	"net/http"

	"github.com/printdreams/cryptocash-ton-battery/internal/abuse"
)

func abuseReject(r *http.Request, userID string, ks *abuse.Killswitch, lim *abuse.Limiter) string {
	if ks != nil && ks.Paused(r.Context()) {
		return "battery-paused"
	}
	if lim != nil {
		if ok, reason := lim.Allow(userID); !ok {
			return reason
		}
	}
	return ""
}

func rejectAbuse(w http.ResponseWriter, reason string) {
	w.Header().Set("Supported-By-Battery", "true")
	w.Header().Set("Allowed-By-Battery", "false")
	w.Header().Set("Reject-Reason", reason)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"allowed":      false,
		"supported":    true,
		"rejectReason": reason,
	})
}
