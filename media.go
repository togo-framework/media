// Package media is a media library for togo — upload files to the storage
// plugin, attach them to any model, and generate image conversions (thumbnails).
//
// It is the togo answer to Spatie Media Library / Rails Active Storage: blobs
// live in the storage plugin (k.Storage); metadata + conversions are recorded so
// you can attach media to a model by (model_type, model_id, collection) and
// fetch it back with derived sizes.
//
//	m, _ := media.FromKernel(k)
//	rec, _ := m.Attach(ctx, "User", userID, "avatar", "me.png", bytes)
//	url := m.URL(rec, "thumb")
package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/togo-framework/togo"
)

// Media is a stored file attached to a model.
type Media struct {
	ID          string            `json:"id"`
	ModelType   string            `json:"model_type"`
	ModelID     string            `json:"model_id"`
	Collection  string            `json:"collection"`
	Disk        string            `json:"disk"`
	Path        string            `json:"path"`
	Name        string            `json:"name"`
	Mime        string            `json:"mime"`
	Size        int64             `json:"size"`
	Conversions map[string]string `json:"conversions,omitempty"` // conversion name → storage path
	Custom      map[string]any    `json:"custom,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Conversion is a named image transformation applied on attach.
type Conversion struct {
	Name   string
	Width  int
	Height int
	Fit    bool // true = Fit (preserve aspect within WxH); false = Thumbnail (crop to fill)
}

// DefaultConversions are generated for image uploads unless overridden.
var DefaultConversions = []Conversion{
	{Name: "thumb", Width: 200, Height: 200, Fit: false},
	{Name: "preview", Width: 1024, Height: 1024, Fit: true},
}

// Service is the media runtime stored on the kernel (k.Get("media")).
type Service struct {
	k           *togo.Kernel
	store       Store
	storage     togo.Storage
	conversions []Conversion
}

func init() {
	togo.RegisterProviderFunc("media", togo.PriorityLate+20, func(k *togo.Kernel) error {
		s := &Service{k: k, storage: k.Storage, conversions: DefaultConversions}
		if st, err := newSQLStore(k); err == nil {
			s.store = st // persistent metadata when a DB is reachable
		} else {
			s.store = newMemStore() // in-memory fallback
		}
		k.Set("media", s)
		s.routes()
		return nil
	})
}

// FromKernel returns the media Service registered on the kernel.
func FromKernel(k *togo.Kernel) (*Service, bool) {
	v, ok := k.Get("media")
	if !ok {
		return nil, false
	}
	s, ok := v.(*Service)
	return s, ok
}

// SetConversions overrides the image conversions generated on attach.
func (s *Service) SetConversions(c []Conversion) { s.conversions = c }

// Attach stores file bytes in the storage plugin, records the media row against
// (modelType, modelID, collection), and generates image conversions when the
// upload is an image.
func (s *Service) Attach(ctx context.Context, modelType, modelID, collection, name string, data []byte) (*Media, error) {
	if s.storage == nil {
		return nil, errors.New("media: no storage configured — install a storage plugin (e.g. togo install togo-framework/storage-s3)")
	}
	id := newID()
	mime := http.DetectContentType(data)
	base := path.Join("media", safe(modelType), safe(modelID), safe(collection))
	p := path.Join(base, id+"-"+safe(name))
	if err := s.storage.Put(p, data); err != nil {
		return nil, err
	}
	m := &Media{
		ID: id, ModelType: modelType, ModelID: modelID, Collection: collection,
		Disk: "default", Path: p, Name: name, Mime: mime, Size: int64(len(data)),
		Conversions: map[string]string{}, CreatedAt: time.Now().UTC(),
	}
	if isImage(mime) {
		for _, c := range s.conversions {
			conv, err := generateConversion(data, c)
			if err != nil {
				continue // graceful — skip a conversion that fails
			}
			cp := path.Join(base, id+"-"+c.Name+".jpg")
			if err := s.storage.Put(cp, conv); err == nil {
				m.Conversions[c.Name] = cp
			}
		}
	}
	if err := s.store.Save(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Get returns the media attached to a model in a collection.
func (s *Service) Get(ctx context.Context, modelType, modelID, collection string) ([]*Media, error) {
	return s.store.List(ctx, modelType, modelID, collection)
}

// Find returns a single media record by id.
func (s *Service) Find(ctx context.Context, id string) (*Media, error) {
	return s.store.Get(ctx, id)
}

// Bytes reads the stored bytes for a media record (optionally a conversion).
func (s *Service) Bytes(m *Media, conversion string) ([]byte, error) {
	if s.storage == nil {
		return nil, errors.New("media: no storage configured")
	}
	p := m.Path
	if conversion != "" {
		if cp, ok := m.Conversions[conversion]; ok {
			p = cp
		}
	}
	return s.storage.Get(p)
}

// Delete removes a media record and its stored blobs (original + conversions).
func (s *Service) Delete(ctx context.Context, id string) error {
	m, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.storage != nil {
		_ = s.storage.Delete(m.Path)
		for _, cp := range m.Conversions {
			_ = s.storage.Delete(cp)
		}
	}
	return s.store.Delete(ctx, id)
}

// URL returns the route that serves a media record (or a named conversion).
func (s *Service) URL(m *Media, conversion string) string {
	if conversion != "" {
		if _, ok := m.Conversions[conversion]; ok {
			return "/api/media/" + m.ID + "/" + conversion
		}
	}
	return "/api/media/" + m.ID
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isImage(mime string) bool { return strings.HasPrefix(mime, "image/") }

// safe sanitizes a path segment to alphanumerics, dot, dash and underscore.
func safe(s string) string {
	if s == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
