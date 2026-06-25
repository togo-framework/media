package media

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func parseTime(s string) (time.Time, error) { return time.Parse(rfc3339, s) }

// routes mounts the media REST API on the kernel router.
func (s *Service) routes() {
	if s.k == nil || s.k.Router == nil {
		return
	}
	s.k.Router.Route("/api/media", func(r chi.Router) {
		r.Post("/", s.handleUpload)               // multipart: model_type, model_id, collection, file
		r.Get("/", s.handleList)                  // ?model_type=&model_id=&collection=
		r.Get("/{id}", s.handleServe)             // stream original
		r.Get("/{id}/{conversion}", s.handleServe) // stream a conversion
		r.Delete("/{id}", s.handleDelete)
	})
}

func (s *Service) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		httpErr(w, http.StatusBadRequest, err)
		return
	}
	m, err := s.Attach(r.Context(), r.FormValue("model_type"), r.FormValue("model_id"), r.FormValue("collection"), hdr.Filename, data)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list, err := s.Get(r.Context(), q.Get("model_type"), q.Get("model_id"), q.Get("collection"))
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Service) handleServe(w http.ResponseWriter, r *http.Request) {
	m, err := s.Find(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpErr(w, http.StatusNotFound, err)
		return
	}
	conv := chi.URLParam(r, "conversion")
	data, err := s.Bytes(m, conv)
	if err != nil {
		httpErr(w, http.StatusNotFound, err)
		return
	}
	ct := m.Mime
	if conv != "" {
		ct = "image/jpeg"
	}
	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(data)
}

func (s *Service) handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
