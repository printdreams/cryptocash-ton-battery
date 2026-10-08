package handler

import "net/http"

// HealthCheck godoc
// @Summary      Check service status
// @Description  Returns OK status if the API is running
// @Tags         system
// @Produce      json
// @Success      200  {object}  handler.HealthResponse
// @Router       /health [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"OK","message":"TON Battery API works!"}`))
}
