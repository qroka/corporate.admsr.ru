package handlers

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/httpx"
)

type OFO struct {
	Pool *pgxpool.Pool
}

func (h *OFO) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w)
		return
	}
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, title, parent, type, sort_order
		FROM public.ofo
		ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "Query error",
		})
		return
	}
	defer rows.Close()

	var data []map[string]any
	for rows.Next() {
		var id, parent, sortOrder int64
		var title, typ string
		if err := rows.Scan(&id, &title, &parent, &typ, &sortOrder); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "Query error",
			})
			return
		}
		data = append(data, map[string]any{
			"id":         id,
			"title":      title,
			"parent":     parent,
			"type":       typ,
			"sort_order": sortOrder,
		})
	}
	if data == nil {
		data = []map[string]any{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *OFO) Seats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w)
		return
	}
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, title, ofo, insurance, rating
		FROM public.ofo_seats
		ORDER BY id ASC`)
	if err != nil {
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"message": "Query error",
		})
		return
	}
	defer rows.Close()

	var data []map[string]any
	for rows.Next() {
		var id int64
		var title, ofo, insurance, rating string
		if err := rows.Scan(&id, &title, &ofo, &insurance, &rating); err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"message": "Query error",
			})
			return
		}
		data = append(data, map[string]any{
			"id":        id,
			"title":     title,
			"ofo":       ofo,
			"insurance": insurance,
			"rating":    rating,
		})
	}
	if data == nil {
		data = []map[string]any{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func (h *OFO) Tree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w)
		return
	}
	ctx := r.Context()

	catRows, err := h.Pool.Query(ctx, `
		SELECT id, name, sort_order FROM public.ofo_category ORDER BY sort_order, id`)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer catRows.Close()

	categories := []map[string]any{}
	for catRows.Next() {
		var id, sortOrder int64
		var name string
		if err := catRows.Scan(&id, &name, &sortOrder); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		categories = append(categories, map[string]any{
			"id":         id,
			"name":       name,
			"sort_order": sortOrder,
		})
	}

	unitRows, err := h.Pool.Query(ctx, `
		SELECT id, name, category_id, parent_id, level, unit_number, family_number, sort_order
		FROM public.ofo_unit
		ORDER BY category_id, level, name`)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer unitRows.Close()

	posCounts := map[int64]int{}
	posRows, err := h.Pool.Query(ctx, `
		SELECT unit_number, COUNT(*)::int FROM public.ofo_unit_position GROUP BY unit_number`)
	if err == nil {
		for posRows.Next() {
			var unitNumber int64
			var count int
			if err := posRows.Scan(&unitNumber, &count); err == nil {
				posCounts[unitNumber] = count
			}
		}
		posRows.Close()
	}

	userCounts := map[int64]int{}
	userRows, err := h.Pool.Query(ctx, `
		SELECT ofo, COUNT(*)::int FROM public.user_info WHERE ofo ~ '^[0-9]+$' GROUP BY ofo`)
	if err == nil {
		for userRows.Next() {
			var ofoID int64
			var count int
			var ofoStr string
			if err := userRows.Scan(&ofoStr, &count); err == nil {
				if n, err := strconv.ParseInt(ofoStr, 10, 64); err == nil {
					ofoID = n
					userCounts[ofoID] = count
				}
			}
		}
		userRows.Close()
	}

	units := []map[string]any{}
	for unitRows.Next() {
		var id, categoryID, level, unitNumber, sortOrder int64
		var parentID, familyNumber *int64
		var name string
		if err := unitRows.Scan(&id, &name, &categoryID, &parentID, &level, &unitNumber, &familyNumber, &sortOrder); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		units = append(units, map[string]any{
			"id":             id,
			"name":           name,
			"category_id":    categoryID,
			"parent_id":      parentID,
			"level":          level,
			"unit_number":    unitNumber,
			"family_number":  familyNumber,
			"sort_order":     sortOrder,
			"position_count": posCounts[unitNumber],
			"user_count":     userCounts[id],
		})
	}

	httpx.OK(w, map[string]any{
		"categories": categories,
		"units":      units,
	}, "OK")
}

func (h *OFO) Positions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.MethodNotAllowed(w)
		return
	}
	unitNumber, err := strconv.ParseInt(r.URL.Query().Get("unit_number"), 10, 64)
	if err != nil || unitNumber <= 0 {
		httpx.Fail(w, http.StatusBadRequest, "Укажите unit_number")
		return
	}

	rows, err := h.Pool.Query(r.Context(), `
		SELECT p.id, p.name, p.is_head
		FROM public.ofo_position p
		JOIN public.ofo_unit_position oup ON oup.position_id = p.id
		WHERE oup.unit_number = $1
		ORDER BY p.is_head DESC, p.sort_order, p.name`, unitNumber)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	defer rows.Close()

	data := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		var isHead bool
		if err := rows.Scan(&id, &name, &isHead); err != nil {
			httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
			return
		}
		data = append(data, map[string]any{
			"id":      id,
			"name":    name,
			"is_head": isHead,
		})
	}
	httpx.OK(w, data, "OK")
}
