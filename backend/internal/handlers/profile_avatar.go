package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"corporate.admsr.ru/backend/internal/auth"
	"corporate.admsr.ru/backend/internal/httpx"
	"corporate.admsr.ru/backend/internal/media"
)

// ProfileAvatar — загрузка собственного аватара (/api/profile_avatar.php, POST
// multipart, поле `avatar`). Только для вошедшего и только для себя: личность
// берётся из сессии. Картинка обрезается по центру до квадрата 512×512 и сразу
// становится аватаром; файл прежнего загруженного аватара удаляется.
type ProfileAvatar struct {
	Pool      *pgxpool.Pool
	Auth      *auth.Service
	UploadDir string
}

const avatarMaxBytes = 5 << 20

func (h *ProfileAvatar) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.MethodNotAllowed(w)
		return
	}
	cur, ok := requireUser(w, r, h.Auth)
	if !ok {
		return
	}

	// Запас на служебные поля multipart; сам файл проверяем ниже.
	r.Body = http.MaxBytesReader(w, r.Body, avatarMaxBytes+(1<<20))
	if err := r.ParseMultipartForm(avatarMaxBytes); err != nil {
		httpx.Fail(w, http.StatusRequestEntityTooLarge, "Файл больше 5 МБ")
		return
	}
	file, _, err := r.FormFile("avatar")
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Файл не передан")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, avatarMaxBytes+1))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "Ошибка загрузки файла")
		return
	}
	if len(raw) > avatarMaxBytes {
		httpx.Fail(w, http.StatusRequestEntityTooLarge, "Файл больше 5 МБ")
		return
	}
	if mime := media.DetectMIME(raw); mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		httpx.Fail(w, http.StatusUnsupportedMediaType, "Допустимы только JPEG, PNG или WebP")
		return
	}

	root := media.ImgRoot(h.UploadDir)
	url, err := media.SaveAvatar(root, cur.ID, raw)
	if err != nil {
		log.Printf("profile avatar: user %d: %v", cur.ID, err)
		httpx.Fail(w, http.StatusUnprocessableEntity, "Не удалось обработать изображение — попробуйте другой файл")
		return
	}

	var old string
	if err := h.Pool.QueryRow(r.Context(),
		`SELECT COALESCE(avatar_url, '') FROM public.user_info WHERE id = $1`, cur.ID).Scan(&old); err != nil {
		media.RemoveUploadedAvatar(root, cur.ID, url)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Fail(w, http.StatusNotFound, "Пользователь не найден")
			return
		}
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	if _, err := h.Pool.Exec(r.Context(), `UPDATE public.user_info SET avatar_url = $1 WHERE id = $2`, url, cur.ID); err != nil {
		media.RemoveUploadedAvatar(root, cur.ID, url)
		httpx.Fail(w, http.StatusInternalServerError, "Ошибка подключения к БД")
		return
	}
	media.RemoveUploadedAvatar(root, cur.ID, old)
	httpx.OK(w, map[string]any{"avatar_url": url}, "Фото загружено")
}
