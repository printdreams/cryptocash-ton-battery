package handler

import (
	"net/http"

	"github.com/printdreams/cryptocash-ton-battery/internal/catalog"
)

type CatalogHandler struct{}

func NewCatalogHandler() *CatalogHandler {
	return &CatalogHandler{}
}

// Products godoc
// @Summary      Charge packs catalog
// @Description  Returns the available charge packs (mock catalog until real IAP)
// @Tags         charges
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /products [get]
func (h *CatalogHandler) Products(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, map[string]interface{}{"products": catalog.All()})
}
