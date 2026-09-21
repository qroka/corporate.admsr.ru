package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
)

var monthNamesRU = map[int]string{
	1: "январь", 2: "февраль", 3: "март", 4: "апрель", 5: "май", 6: "июнь",
	7: "июль", 8: "август", 9: "сентябрь", 10: "октябрь", 11: "ноябрь", 12: "декабрь",
}

type Birthdays struct {
	Pool      *pgxpool.Pool
	Auth      *auth.Service
	UploadDir string
}

func (h *Birthdays) dir() string {
	return h.resolveBirthdaysDir()
}

func (h *Birthdays) oldDir() string {
	// рядом с выбранным birthdays_xlsx
	return filepath.Join(filepath.Dir(h.dir()), "birthdays_xlsx_old")
}

func (h *Birthdays) manifestPath() string {
	return filepath.Join(h.dir(), "manifest.json")
}

// resolveBirthdaysDir выбирает каталог с реальными xlsx (UPLOAD_DIR может
// указывать не туда, где лежат файлы PHP/админки).
func (h *Birthdays) resolveBirthdaysDir() string {
	var candidates []string
	if v := strings.TrimSpace(os.Getenv("BIRTHDAYS_DIR")); v != "" {
		candidates = append(candidates, v)
	}
	upload := strings.TrimSpace(h.UploadDir)
	if upload != "" {
		candidates = append(candidates, filepath.Join(upload, "birthdays_xlsx"))
	}
	candidates = append(candidates,
		"/var/lib/corporate-app/uploads/birthdays_xlsx",
		"/var/www/corporate.admsr.ru/public/birthdays_xlsx",
		"public/birthdays_xlsx",
	)

	var fallback string
	for _, d := range candidates {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if fallback == "" {
			fallback = d
		}
		matches, _ := filepath.Glob(filepath.Join(d, "*.xlsx"))
		n := 0
		for _, m := range matches {
			if strings.EqualFold(filepath.Base(m), ".gitkeep") {
				continue
			}
			if strings.HasSuffix(strings.ToLower(m), ".xlsx") {
				n++
			}
		}
		if n > 0 {
			return d
		}
		if st, err := os.Stat(filepath.Join(d, "manifest.json")); err == nil && st.Size() > 2 {
			return d
		}
	}
	if fallback != "" {
		return fallback
	}
	return "birthdays_xlsx"
}

func (h *Birthdays) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		httpx.Fail(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
	}
}

func (h *Birthdays) handleGet(w http.ResponseWriter, r *http.Request) {
	dir := h.dir()
	if r.URL.Query().Get("debug") != "" {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.xlsx"))
		info, err := os.Stat(dir)
		parseErrs := map[string]string{}
		entryCounts := map[string]int{}
		for _, file := range matches {
			parsed, perr := parseBirthdayFile(file)
			base := filepath.Base(file)
			if perr != nil {
				parseErrs[base] = perr.Error()
				continue
			}
			if parsed == nil {
				parseErrs[base] = "nil parse"
				continue
			}
			entryCounts[base] = len(parsed.Entries)
			if len(parsed.Entries) == 0 {
				parseErrs[base] = "0 entries"
			}
		}
		httpx.OK(w, map[string]any{
			"uploadDir":    h.UploadDir,
			"dir":          dir,
			"manifestPath": h.manifestPath(),
			"dirExists":    err == nil && info.IsDir(),
			"xlsx":         matches,
			"entryCounts":  entryCounts,
			"parseErrors":  parseErrs,
			"manifest":     h.loadManifest(),
		}, "OK")
		return
	}
	if r.URL.Query().Get("manifest") != "" {
		httpx.OK(w, h.loadManifest(), "OK")
		return
	}
	avatarMap := h.buildAvatarMap(r.Context())
	all := make([]map[string]any, 0)
	matches, _ := filepath.Glob(filepath.Join(dir, "*.xlsx"))
	sort.Strings(matches)
	for _, file := range matches {
		parsed, err := parseBirthdayFile(file)
		if err != nil || parsed == nil {
			continue
		}
		for _, e := range parsed.Entries {
			row := map[string]any{
				"fio":   e.FIO,
				"month": e.Month,
				"day":   e.Day,
			}
			if av := avatarMap[normFIO(e.FIO)]; av != "" {
				row["avatar"] = av
			} else {
				row["avatar"] = nil
			}
			all = append(all, row)
		}
	}
	httpx.OK(w, all, "OK")
}

func (h *Birthdays) handlePost(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSection(w, r, h.Pool, h.Auth, "birthdays"); !ok {
		return
	}
	if err := os.MkdirAll(h.dir(), 0o755); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось создать каталог")
		return
	}
	if err := os.MkdirAll(h.oldDir(), 0o755); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось создать каталог")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if r.ContentLength > 0 {
			httpx.Fail(w, http.StatusRequestEntityTooLarge, "Файл слишком большой — превышен лимит сервера (post_max_size).")
			return
		}
		httpx.Fail(w, http.StatusBadRequest, "Файл не передан")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Файл не передан")
		return
	}
	defer file.Close()

	origName := header.Filename
	if !strings.HasSuffix(strings.ToLower(origName), ".xlsx") {
		httpx.Fail(w, http.StatusBadRequest, "Допустим только файл .xlsx")
		return
	}

	tmp, err := os.CreateTemp("", "bday-*.xlsx")
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	tmp.Close()

	parsed, err := parseBirthdayFile(tmpPath)
	if err != nil || parsed == nil {
		httpx.Fail(w, http.StatusBadRequest, "Не удалось прочитать xlsx-файл")
		return
	}
	month := parsed.Month
	if month < 1 || month > 12 {
		httpx.Fail(w, http.StatusUnprocessableEntity, "Не удалось определить месяц (проверьте ячейку A1)")
		return
	}

	nn := fmt.Sprintf("%02d", month)
	target := filepath.Join(h.dir(), nn+".xlsx")
	if _, err := os.Stat(target); err == nil {
		manifest := h.loadManifest()
		prevName := nn + ".xlsx"
		if m, ok := manifest[strconv.Itoa(month)].(map[string]any); ok {
			if fn, ok := m["filename"].(string); ok && fn != "" {
				prevName = fn
			}
		}
		safePrev := regexp.MustCompile(`[^\p{L}\p{N}._-]+`).ReplaceAllString(prevName, "_")
		oldName := fmt.Sprintf("%s_%s_%s", nn, time.Now().Format("20060102_150405"), safePrev)
		if !strings.HasSuffix(strings.ToLower(oldName), ".xlsx") {
			oldName += ".xlsx"
		}
		_ = os.Rename(target, filepath.Join(h.oldDir(), oldName))
	}

	in, err := os.Open(tmpPath)
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	out, err := os.Create(target)
	if err != nil {
		in.Close()
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	if _, err := io.Copy(out, in); err != nil {
		in.Close()
		out.Close()
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	in.Close()
	out.Close()

	manifest := h.loadManifest()
	manifest[strconv.Itoa(month)] = map[string]any{
		"filename":    origName,
		"year":        parsed.Year,
		"count":       len(parsed.Entries),
		"uploaded_at": time.Now().Format(time.RFC3339),
	}
	if err := h.saveManifest(manifest); err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "Не удалось сохранить manifest")
		return
	}
	msg := fmt.Sprintf("Файл «%s» загружен", monthNamesRU[month])
	httpx.OK(w, manifest, msg)
}

type birthdayEntry struct {
	FIO   string
	Month int
	Day   int
}

type birthdayParsed struct {
	Month   int
	Year    int
	Entries []birthdayEntry
}

func parseBirthdayFile(path string) (*birthdayParsed, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("empty sheet")
	}
	sheet := sheets[0]

	rows, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil || len(rows) == 0 {
		rows, err = f.GetRows(sheet)
		if err != nil {
			return nil, fmt.Errorf("no rows: %w", err)
		}
	}

	headerA, _ := f.GetCellValue(sheet, "A1", excelize.Options{RawCellValue: true})
	headerB, _ := f.GetCellValue(sheet, "B1", excelize.Options{RawCellValue: true})
	if strings.TrimSpace(headerA) == "" && len(rows) > 0 && len(rows[0]) > 0 {
		headerA = rows[0][0]
	}
	if strings.TrimSpace(headerB) == "" && len(rows) > 0 && len(rows[0]) > 1 {
		headerB = rows[0][1]
	}
	month := monthNameToNumber(headerA)
	year, _ := strconv.Atoi(strings.TrimSpace(headerB))

	entries := collectBirthdayEntries(f, sheet, rows)
	if len(entries) == 0 {
		// GetRows иногда пропускает ячейки с датами — читаем A/B по номерам строк
		entries = collectBirthdayEntriesByCells(f, sheet, 500)
	}

	if month == 0 && len(entries) > 0 {
		counts := map[int]int{}
		for _, e := range entries {
			counts[e.Month]++
		}
		best, bestN := 0, 0
		for m, n := range counts {
			if n > bestN {
				best, bestN = m, n
			}
		}
		month = best
	}

	return &birthdayParsed{Month: month, Year: year, Entries: entries}, nil
}

func collectBirthdayEntries(f *excelize.File, sheet string, rows [][]string) []birthdayEntry {
	var entries []birthdayEntry
	for i := 1; i < len(rows); i++ {
		cells := rows[i]
		fio := ""
		if len(cells) > 0 {
			fio = strings.TrimSpace(cells[0])
		}
		var rawDate any
		if len(cells) > 1 {
			rawDate = cells[1]
		}
		if rawDate == nil || strings.TrimSpace(fmt.Sprint(rawDate)) == "" {
			axis, _ := excelize.CoordinatesToCellName(2, i+1)
			if v, err := f.GetCellValue(sheet, axis, excelize.Options{RawCellValue: true}); err == nil && v != "" {
				rawDate = v
			} else if v, err := f.GetCellValue(sheet, axis); err == nil {
				rawDate = v
			}
		}
		md := excelToMD(rawDate)
		if fio == "" || md == nil {
			continue
		}
		entries = append(entries, birthdayEntry{FIO: fio, Month: md.M, Day: md.D})
	}
	return entries
}

func collectBirthdayEntriesByCells(f *excelize.File, sheet string, maxRows int) []birthdayEntry {
	var entries []birthdayEntry
	emptyStreak := 0
	for row := 2; row <= maxRows; row++ {
		axisA, _ := excelize.CoordinatesToCellName(1, row)
		axisB, _ := excelize.CoordinatesToCellName(2, row)
		fio, _ := f.GetCellValue(sheet, axisA)
		fio = strings.TrimSpace(fio)
		raw, _ := f.GetCellValue(sheet, axisB, excelize.Options{RawCellValue: true})
		if raw == "" {
			raw, _ = f.GetCellValue(sheet, axisB)
		}
		if fio == "" {
			emptyStreak++
			if emptyStreak >= 15 {
				break
			}
			continue
		}
		emptyStreak = 0
		md := excelToMD(raw)
		if md == nil {
			continue
		}
		entries = append(entries, birthdayEntry{FIO: fio, Month: md.M, Day: md.D})
	}
	return entries
}

type mdPair struct {
	M, D int
}

func excelToMD(raw any) *mdPair {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		if n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64); err == nil {
			return serialToMD(n)
		}
		layouts := []string{
			"2006-01-02",
			"02.01.2006",
			"2.01.2006",
			"02.1.2006",
			"2.1.2006",
			"02.01.06",
			"2.1.06",
			"01/02/2006",
			"1/2/2006",
			"02-01-2006",
			"2006/01/02",
			time.RFC3339,
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, s); err == nil {
				return &mdPair{M: int(t.Month()), D: t.Day()}
			}
		}
		// «15 июня 1990» / «15 июня» — день + русский месяц
		if md := parseRussianDate(s); md != nil {
			return md
		}
		return nil
	case float64:
		return serialToMD(v)
	case int:
		return serialToMD(float64(v))
	case int64:
		return serialToMD(float64(v))
	default:
		return excelToMD(fmt.Sprint(v))
	}
}

func parseRussianDate(s string) *mdPair {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "ё", "е")
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return nil
	}
	day, err := strconv.Atoi(fields[0])
	if err != nil || day < 1 || day > 31 {
		return nil
	}
	m := monthNameToNumber(fields[1])
	if m == 0 {
		return nil
	}
	return &mdPair{M: m, D: day}
}

func serialToMD(serial float64) *mdPair {
	n := int(serial + 0.5)
	if n <= 0 {
		return nil
	}
	unix := int64(n-25569) * 86400
	t := time.Unix(unix, 0).UTC()
	return &mdPair{M: int(t.Month()), D: t.Day()}
}

func monthNameToNumber(raw string) int {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return 0
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= 12 {
		return n
	}
	stems := map[int]string{
		1: "янв", 2: "фев", 3: "мар", 4: "апр", 5: "ма", 6: "июн",
		7: "июл", 8: "авг", 9: "сен", 10: "окт", 11: "ноя", 12: "дек",
	}
	order := []int{1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12, 5}
	for _, m := range order {
		if strings.Contains(s, stems[m]) {
			return m
		}
	}
	return 0
}

func normFIO(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "ё", "е")
	return strings.Join(strings.Fields(s), " ")
}

func (h *Birthdays) buildAvatarMap(ctx context.Context) map[string]string {
	out := map[string]string{}
	rows, err := h.Pool.Query(ctx, `SELECT surname, firstname, lastname, avatar_url FROM public.user_info`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var surname, firstname, lastname, avatarURL *string
		if err := rows.Scan(&surname, &firstname, &lastname, &avatarURL); err != nil {
			continue
		}
		av := ""
		if avatarURL != nil {
			av = strings.TrimSpace(*avatarURL)
		}
		if av == "" {
			continue
		}
		full := normFIO(joinNames(surname, firstname, lastname))
		if full != "" {
			if _, ok := out[full]; !ok {
				out[full] = av
			}
		}
	}
	return out
}

func joinNames(parts ...*string) string {
	var b strings.Builder
	for _, p := range parts {
		if p == nil {
			continue
		}
		s := strings.TrimSpace(*p)
		if s == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s)
	}
	return b.String()
}

func (h *Birthdays) loadManifest() map[string]any {
	data, err := os.ReadFile(h.manifestPath())
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil || m == nil {
		return map[string]any{}
	}
	return m
}

func (h *Birthdays) saveManifest(m map[string]any) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.manifestPath(), data, 0o644)
}
