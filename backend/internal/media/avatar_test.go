package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 30, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSaveAvatarCropsToSquare(t *testing.T) {
	root := t.TempDir()
	url, err := SaveAvatar(root, 42, testPNG(t, 1200, 300))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, AvatarsUploadsURL+"42-") || !strings.HasSuffix(url, ".jpg") {
		t.Fatalf("неожиданный URL: %s", url)
	}
	f, err := os.Open(filepath.Join(root, "FullPic", "avatars", "uploads", strings.TrimPrefix(url, AvatarsUploadsURL)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 512 || cfg.Height != 512 {
		t.Fatalf("размер %dx%d, ожидалось 512x512", cfg.Width, cfg.Height)
	}
}

func TestSaveAvatarRejectsGarbage(t *testing.T) {
	if _, err := SaveAvatar(t.TempDir(), 1, []byte("not an image")); err == nil {
		t.Fatal("не-картинка должна отклоняться")
	}
}

// Удаляется только файл этого же пользователя в папке загрузок — остальное
// (стандартный аватар, чужой файл, выход за каталог) не трогается.
func TestRemoveUploadedAvatarOnlyOwnUpload(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "FullPic", "avatars", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(dir, "7-aaaa.jpg")
	other := filepath.Join(dir, "8-bbbb.jpg")
	preset := filepath.Join(root, "FullPic", "avatars", "Alien.png")
	for _, p := range []string{own, other, preset} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	RemoveUploadedAvatar(root, 7, "/img/FullPic/avatars/Alien.png")               // стандартный
	RemoveUploadedAvatar(root, 7, AvatarsUploadsURL+"8-bbbb.jpg")                 // чужой
	RemoveUploadedAvatar(root, 7, AvatarsUploadsURL+"7-../../Alien.png")          // обход каталога
	RemoveUploadedAvatar(root, 7, "https://example.test"+AvatarsUploadsURL+"7-a") // внешний адрес
	for _, p := range []string{own, other, preset} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("файл %s удалён зря", p)
		}
	}

	RemoveUploadedAvatar(root, 7, AvatarsUploadsURL+"7-aaaa.jpg")
	if _, err := os.Stat(own); !os.IsNotExist(err) {
		t.Fatal("собственный загруженный аватар должен удаляться")
	}
}
