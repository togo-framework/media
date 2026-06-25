# media — usage

## Attach media to a model

```go
m, _ := media.FromKernel(k)
rec, err := m.Attach(ctx, "User", userID, "avatar", "me.png", fileBytes)
```

`(model_type, model_id, collection)` scopes media to a model. Image uploads get
conversions generated automatically.

## Conversions

Defaults: `thumb` (200×200, crop-to-fill) and `preview` (1024×1024, fit). Override:

```go
m.SetConversions([]media.Conversion{
    {Name: "thumb", Width: 150, Height: 150, Fit: false},
    {Name: "hero",  Width: 1600, Height: 900, Fit: true},
})
```

`Fit: true` preserves aspect ratio within WxH; `Fit: false` crops to an exact
WxH thumbnail.

## Fetch & serve

```go
items, _ := m.Get(ctx, "User", userID, "avatar") // []*Media
data,  _ := m.Bytes(rec, "thumb")                // raw bytes
url := m.URL(rec, "thumb")                        // /api/media/<id>/thumb
_ = m.Delete(ctx, rec.ID)                          // removes blobs + conversions
```

## REST API

| Method | Path |
|---|---|
| `POST` | `/api/media` (multipart: `model_type`, `model_id`, `collection`, `file`) |
| `GET` | `/api/media?model_type=&model_id=&collection=` |
| `GET` | `/api/media/{id}` · `/api/media/{id}/{conversion}` |
| `DELETE` | `/api/media/{id}` |

## Requirements

- A **storage** provider for blobs (`togo install togo-framework/storage-s3`).
- Optional DB for persistent metadata (auto-detected; in-memory otherwise).
