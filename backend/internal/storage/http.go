package storage

import (
	"net/http"
	"path"
	"strings"
)

func Files(blobs Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.PathValue("key"), "/")
		if !strings.HasPrefix(key, "mail-images/") {
			http.NotFound(w, r)
			return
		}
		b, err := blobs.Get(r.Context(), key)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType(key))
		w.Header().Set("Cache-Control", "private, max-age=86400")
		_, _ = w.Write(b)
	}
}

func contentType(key string) string {
	switch strings.ToLower(path.Ext(key)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
