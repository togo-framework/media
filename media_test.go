package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/disintegration/imaging"
)

// fakeStorage is an in-memory togo.Storage for tests.
type fakeStorage struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newFakeStorage() *fakeStorage { return &fakeStorage{m: map[string][]byte{}} }

func (f *fakeStorage) Put(p string, d []byte) error {
	f.mu.Lock()
	f.m[p] = append([]byte(nil), d...)
	f.mu.Unlock()
	return nil
}
func (f *fakeStorage) Get(p string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.m[p]
	if !ok {
		return nil, os.ErrNotExist
	}
	return d, nil
}
func (f *fakeStorage) Delete(p string) error {
	f.mu.Lock()
	delete(f.m, p)
	f.mu.Unlock()
	return nil
}
func (f *fakeStorage) Path(p string) string { return p }

func testService() *Service {
	return &Service{storage: newFakeStorage(), store: newMemStore(), conversions: DefaultConversions}
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestAttachImageWithConversions(t *testing.T) {
	s := testService()
	data := pngBytes(400, 300)
	m, err := s.Attach(context.Background(), "User", "u1", "avatar", "pic.png", data)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == "" || m.ModelType != "User" || m.ModelID != "u1" || m.Size != int64(len(data)) {
		t.Fatalf("bad media record: %+v", m)
	}
	if !strings.HasPrefix(m.Mime, "image/") {
		t.Fatalf("mime = %q, want image/*", m.Mime)
	}
	for _, name := range []string{"thumb", "preview"} {
		if _, ok := m.Conversions[name]; !ok {
			t.Fatalf("missing %q conversion: %v", name, m.Conversions)
		}
	}
	// The thumb must be a valid 200x200 JPEG.
	tb, err := s.Bytes(m, "thumb")
	if err != nil {
		t.Fatal(err)
	}
	img, err := imaging.Decode(bytes.NewReader(tb))
	if err != nil {
		t.Fatalf("thumb not decodable: %v", err)
	}
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 200 {
		t.Errorf("thumb size = %v, want 200x200", img.Bounds())
	}
}

func TestAttachNonImageSkipsConversions(t *testing.T) {
	s := testService()
	m, err := s.Attach(context.Background(), "Doc", "d1", "files", "notes.txt", []byte("hello world, this is plain text content"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Conversions) != 0 {
		t.Fatalf("non-image should have no conversions, got %v", m.Conversions)
	}
}

func TestGetScoping(t *testing.T) {
	s := testService()
	ctx := context.Background()
	_, _ = s.Attach(ctx, "User", "u1", "avatar", "a.png", pngBytes(50, 50))
	_, _ = s.Attach(ctx, "User", "u1", "gallery", "g.png", pngBytes(50, 50))
	_, _ = s.Attach(ctx, "User", "u2", "avatar", "b.png", pngBytes(50, 50))

	if av, _ := s.Get(ctx, "User", "u1", "avatar"); len(av) != 1 {
		t.Fatalf("u1 avatar count = %d, want 1", len(av))
	}
	if all, _ := s.Get(ctx, "User", "u1", ""); len(all) != 2 {
		t.Fatalf("u1 all count = %d, want 2", len(all))
	}
}

func TestURLAndDelete(t *testing.T) {
	s := testService()
	ctx := context.Background()
	m, _ := s.Attach(ctx, "User", "u1", "avatar", "a.png", pngBytes(50, 50))
	if got := s.URL(m, ""); got != "/api/media/"+m.ID {
		t.Errorf("URL = %q", got)
	}
	if got := s.URL(m, "thumb"); got != "/api/media/"+m.ID+"/thumb" {
		t.Errorf("conversion URL = %q", got)
	}
	if got := s.URL(m, "nope"); got != "/api/media/"+m.ID {
		t.Errorf("unknown conversion should fall back, got %q", got)
	}
	if err := s.Delete(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Find(ctx, m.ID); err == nil {
		t.Error("deleted media still found")
	}
}
