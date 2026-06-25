# media — docs

The `togo-framework/media` plugin: upload files to the storage plugin, attach
them to any model, and generate image conversions/thumbnails.

- **Install:** `togo install togo-framework/media`
- **Marketplace:** https://to-go.dev/plugins/media
- **Source:** https://github.com/togo-framework/media

## Contents
- [Usage](usage.md) — attaching media, conversions, URLs, REST API

Blobs are stored via the togo **storage** plugin; metadata persists in your DB
(table `media`) when reachable, otherwise in memory. See [usage.md](usage.md).
