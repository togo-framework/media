---
name: media
description: Attach uploads to models and generate image conversions/thumbnails in a togo app using the media plugin (Attach, Get, conversions, URLs, REST upload endpoints).
---

# togo media

Use the `togo-framework/media` plugin to handle file uploads + image conversions.

## Attach
```go
m, _ := media.FromKernel(k)
rec, _ := m.Attach(ctx, "User", userID, "avatar", filename, bytes)
```
`(model_type, model_id, collection)` scopes media to a model.

## Conversions
Image uploads auto-generate `thumb` (200×200 crop) + `preview` (1024×1024 fit). Override with `m.SetConversions([]media.Conversion{...})` — `Fit:true` = fit within WxH, `Fit:false` = crop to fill.

## Fetch / serve
- `m.Get(ctx, modelType, modelID, collection)` → `[]*Media`
- `m.URL(rec, "thumb")` → route URL; `m.Bytes(rec, "thumb")` → bytes
- `m.Delete(ctx, id)` removes blobs + conversions

## REST
`POST /api/media` (multipart: model_type, model_id, collection, file) · `GET /api/media?model_type=&model_id=&collection=` · `GET /api/media/{id}[/{conversion}]` · `DELETE /api/media/{id}`

## Requirements
- Install a **storage** provider for blobs (`togo install togo-framework/storage-s3`). Without it, `Attach` errors clearly.
- A DB gives persistent metadata; otherwise it's in-memory.
