package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/asihat/b2b-marketplace/backend/internal/httpx"
	"github.com/asihat/b2b-marketplace/backend/internal/services"
	"github.com/asihat/b2b-marketplace/backend/internal/validate"
)

func (h *Handlers) AdminDashboard(w http.ResponseWriter, r *http.Request) error {
	days := services.DashboardDefaultRange
	if raw := strings.TrimSpace(httpx.Query(r, "days")); raw != "" {
		v := validate.New(map[string]any{"days": raw})
		n, ok := v.Int("days", validate.Num{Required: true})
		if ok {
			valid := false
			for _, d := range services.DashboardRanges {
				if int(n) == d {
					valid = true
				}
			}
			if !valid {
				v.Fail("days", "The selected days is invalid.")
			}
		}
		if err := v.Err(); err != nil {
			return err
		}
		days, _ = strconv.Atoi(raw)
	}

	overview, err := h.app.Dashboard.Overview(r.Context(), days)
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusOK, overview)
	return nil
}
