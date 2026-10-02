package media

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

// DetectMIME returns mime from magic bytes.
func DetectMIME(data []byte) string {
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	if len(data) >= 8 && string(data[0:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png"
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	if len(data) >= 8 && string(data[4:8]) == "ftyp" {
		return "video/mp4"
	}
	if len(data) >= 6 && (string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a") {
		return "image/gif"
	}
	return "application/octet-stream"
}

func RandomName() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SaveWebPPair writes FullPic (max 1920) and SmallPic (max 960) as JPEG.
// Filenames keep .webp suffix for URL compatibility with the PHP pipeline / frontend.
func SaveWebPPair(imgRoot string, raw []byte, mime string) (smallURL, fullURL string, err error) {
	img, err := imaging.Decode(bytes.NewReader(raw), imaging.AutoOrientation(true))
	if err != nil {
		img, _, err = image.Decode(bytes.NewReader(raw))
		if err != nil {
			return "", "", fmt.Errorf("decode %s: %w", mime, err)
		}
	}
	base, err := RandomName()
	if err != nil {
		return "", "", err
	}
	// Keep .webp in URL path for SPA compatibility; content is JPEG (browsers tolerate via Content-Type from static server).
	// Prefer real .jpg to avoid broken MIME on nginx static — frontend only needs the path from API.
	name := base + ".jpg"
	fullDir := filepath.Join(imgRoot, "FullPic")
	smallDir := filepath.Join(imgRoot, "SmallPic")
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		return "", "", err
	}
	if err := writeJPEG(filepath.Join(fullDir, name), resizeMax(img, 1920), 95); err != nil {
		return "", "", err
	}
	if err := writeJPEG(filepath.Join(smallDir, name), resizeMax(img, 960), 76); err != nil {
		return "", "", err
	}
	return "/img/SmallPic/" + name, "/img/FullPic/" + name, nil
}

func SaveMP4(imgRoot string, raw []byte) (smallURL, fullURL string, err error) {
	base, err := RandomName()
	if err != nil {
		return "", "", err
	}
	name := base + ".mp4"
	fullDir := filepath.Join(imgRoot, "FullPic")
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(filepath.Join(fullDir, name), raw, 0o644); err != nil {
		return "", "", err
	}
	url := "/img/FullPic/" + name
	return url, url, nil
}

func ImgRoot(uploadDir string) string {
	uploadDir = strings.TrimRight(uploadDir, `/\`)
	return filepath.Join(uploadDir, "img")
}

func resizeMax(img image.Image, maxW int) image.Image {
	if maxW > 0 && img.Bounds().Dx() > maxW {
		return imaging.Resize(img, maxW, 0, imaging.Lanczos)
	}
	return img
}

func writeJPEG(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}

// AvatarsUploadsURL — публичный префикс загруженных сотрудниками аватаров.
// Лежит внутри /img/FullPic/avatars/ (там же стандартные аватары), поэтому
// отдаётся тем же правилом статики, что и остальные картинки.
const AvatarsUploadsURL = "/img/FullPic/avatars/uploads/"

const (
	avatarSize      = 512
	avatarMaxPixels = 40_000_000 // защита от «бомб»: ~6300×6300
)

// SaveAvatar обрезает картинку по центру до квадрата 512×512 и пишет JPEG
// `<userID>-<случайное>.jpg`. Прозрачность заливается белым — JPEG её не хранит.
// Возвращает публичный URL.
func SaveAvatar(imgRoot string, userID int64, raw []byte) (string, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("decode config: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > avatarMaxPixels {
		return "", fmt.Errorf("image too large: %dx%d", cfg.Width, cfg.Height)
	}
	img, err := imaging.Decode(bytes.NewReader(raw), imaging.AutoOrientation(true))
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	square := imaging.Fill(img, avatarSize, avatarSize, imaging.Center, imaging.Lanczos)
	flat := imaging.New(avatarSize, avatarSize, color.White)
	flat = imaging.Overlay(flat, square, image.Pt(0, 0), 1.0)

	base, err := RandomName()
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d-%s.jpg", userID, base)
	dir := filepath.Join(imgRoot, "FullPic", "avatars", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := writeJPEG(filepath.Join(dir, name), flat, 88); err != nil {
		return "", err
	}
	return AvatarsUploadsURL + name, nil
}

// RemoveUploadedAvatar удаляет файл прежнего загруженного аватара. Трогает
// только `/img/FullPic/avatars/uploads/<userID>-<hex>.jpg` этого же
// пользователя — любой другой URL (стандартный аватар, чужой файл, внешний
// адрес) игнорируется.
func RemoveUploadedAvatar(imgRoot string, userID int64, url string) {
	prefix := AvatarsUploadsURL + fmt.Sprintf("%d-", userID)
	if !strings.HasPrefix(url, prefix) {
		return
	}
	name := strings.TrimPrefix(url, AvatarsUploadsURL)
	if name == "" || strings.ContainsAny(name, `/\`) || !strings.HasSuffix(name, ".jpg") {
		return
	}
	_ = os.Remove(filepath.Join(imgRoot, "FullPic", "avatars", "uploads", name))
}
