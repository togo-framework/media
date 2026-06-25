<div align="center">
  <img src=".github/assets/togo-mark.svg" alt="togo" height="64" />
  <h1>togo-framework/media</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1FC7DC" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/media"><img src="https://pkg.go.dev/badge/github.com/togo-framework/media.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>Media library for <a href="https://to-go.dev">togo</a> — uploads, image conversions & thumbnails, attach media to any model.</strong></p>
</div>

## Install

```bash
togo install togo-framework/media
```

`media` is togo's answer to Spatie Media Library / Rails Active Storage. Files are stored through the togo **storage** plugin (`k.Storage`), metadata is persisted, and you **attach media to any model** by `(model_type, model_id, collection)`. Image uploads get **conversions** (e.g. `thumb`, `preview`) generated automatically.

## How it works

- Blobs live in the **storage** plugin — install one: `togo install togo-framework/storage-s3` (or r2 / gdrive / supabase).
- Metadata is stored in your DB when reachable (table `media`), otherwise in memory.
- On image upload, conversions are generated (default `thumb` 200×200 crop, `preview` 1024×1024 fit) and stored as derived files.
- Media is served back through REST routes (`/api/media/...`).

## Usage

```go
import "github.com/togo-framework/media"

m, _ := media.FromKernel(k)

// Attach an upload to a model.
rec, err := m.Attach(ctx, "User", userID, "avatar", "me.png", fileBytes)

// Fetch a model's media in a collection.
items, _ := m.Get(ctx, "User", userID, "avatar")

// Build URLs (original + conversions).
orig  := m.URL(rec, "")       // /api/media/<id>
thumb := m.URL(rec, "thumb")  // /api/media/<id>/thumb

// Read bytes directly, or delete (removes blobs + conversions).
data, _ := m.Bytes(rec, "thumb")
_ = m.Delete(ctx, rec.ID)
```

### Custom conversions

```go
m.SetConversions([]media.Conversion{
    {Name: "thumb", Width: 150, Height: 150, Fit: false},  // crop to fill
    {Name: "hero",  Width: 1600, Height: 900, Fit: true},  // fit within
})
```

## REST API

| Method | Path | |
|---|---|---|
| `POST` | `/api/media` | multipart upload: `model_type`, `model_id`, `collection`, `file` |
| `GET` | `/api/media?model_type=&model_id=&collection=` | list a model's media |
| `GET` | `/api/media/{id}` | stream the original |
| `GET` | `/api/media/{id}/{conversion}` | stream a conversion (e.g. `thumb`) |
| `DELETE` | `/api/media/{id}` | delete (blobs + conversions) |

## Configuration

No required env. Install a **storage** provider for blobs and (optionally) a database for persistent metadata — `media` auto-detects both. Without a storage plugin, `Attach` returns a clear error; without a DB, metadata is kept in memory.

---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
