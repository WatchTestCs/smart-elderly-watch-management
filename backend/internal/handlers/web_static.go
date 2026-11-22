package handlers

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// Vite 产物 /assets/* 文件名带内容哈希，内容变化即换名，可永久缓存。
	cacheImmutable = "public, max-age=31536000, immutable"
	// /static/* 文件名不带哈希（字体、品牌图），给较长但可被更新的缓存期。
	cacheStatic = "public, max-age=604800"
	// 入口 index.html 必须每次向服务器确认（配合 ETag 命中 304），才能及时拿到新版本。
	cacheRevalidate = "no-cache"

	// 小于该大小的文本压缩收益有限，直接走原始文件。
	gzipMinSize = 1024
)

// 只压缩文本类资源；woff2/png/jpg/webp 等本身已压缩，再压只会浪费 CPU。
var compressibleExt = map[string]bool{
	".html": true, ".js": true, ".mjs": true, ".css": true, ".json": true,
	".svg": true, ".txt": true, ".xml": true, ".webmanifest": true, ".map": true,
}

type gzipEntry struct {
	body        []byte
	etag        string
	contentType string
	ok          bool // false 表示不值得压缩（太小或压缩后更大），回退原始文件
}

// staticHandler 在 http.FileServer 之上增加缓存头与按需 gzip。
// 前端产物编译进二进制后内容不可变，因此压缩结果按路径缓存在内存里，每个文件只压缩一次。
type staticHandler struct {
	fsys  fs.FS
	files http.Handler
	cache sync.Map // 清洗后的文件路径 -> *gzipEntry
}

func newStaticHandler(fsys fs.FS) http.Handler {
	return &staticHandler{fsys: fsys, files: http.FileServer(http.FS(fsys))}
}

func cacheControlFor(urlPath string) string {
	switch {
	case strings.HasPrefix(urlPath, "/assets/"):
		return cacheImmutable
	case strings.HasPrefix(urlPath, "/static/"):
		return cacheStatic
	default:
		return cacheRevalidate
	}
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(name), "gzip") {
			continue
		}
		// 显式 q=0 表示拒绝。
		if _, q, found := strings.Cut(strings.ReplaceAll(params, " ", ""), "q="); found {
			if v, err := strconv.ParseFloat(q, 64); err == nil && v == 0 {
				return false
			}
		}
		return true
	}
	return false
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.files.ServeHTTP(w, r)
		return
	}
	// 路径清洗：path.Clean 会折叠 ".."，FileServer 本身也会拒绝越界，这里只用于命中缓存。
	clean := path.Clean("/" + r.URL.Path)
	w.Header().Set("Cache-Control", cacheControlFor(clean))
	// 同一 URL 随 Accept-Encoding 变化，缓存层必须区分。
	w.Header().Add("Vary", "Accept-Encoding")

	name := strings.TrimPrefix(clean, "/")
	if r.URL.Path == "/" {
		name = "index.html"
	}
	if name == "" || !compressibleExt[strings.ToLower(path.Ext(name))] || !acceptsGzip(r) {
		h.files.ServeHTTP(w, r)
		return
	}
	entry := h.gzipped(name)
	if !entry.ok {
		h.files.ServeHTTP(w, r)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", entry.contentType)
	hd.Set("Content-Encoding", "gzip")
	hd.Set("ETag", entry.etag)
	// ServeContent 负责 If-None-Match -> 304、HEAD 与 Range。
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(entry.body))
}

func (h *staticHandler) gzipped(name string) *gzipEntry {
	if v, ok := h.cache.Load(name); ok {
		return v.(*gzipEntry)
	}
	raw, err := fs.ReadFile(h.fsys, name)
	if err != nil {
		// 不存在的路径来自请求，绝不写入缓存，否则可被用来撑爆内存。
		return &gzipEntry{}
	}
	entry := compress(name, raw)
	actual, _ := h.cache.LoadOrStore(name, entry)
	return actual.(*gzipEntry)
}

func compress(name string, raw []byte) *gzipEntry {
	miss := &gzipEntry{}
	if len(raw) < gzipMinSize {
		return miss
	}
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return miss
	}
	if _, err := zw.Write(raw); err != nil || zw.Close() != nil || buf.Len() >= len(raw) {
		return miss
	}
	ct := mime.TypeByExtension(path.Ext(name))
	if ct == "" {
		ct = "application/octet-stream"
	}
	sum := sha256.Sum256(buf.Bytes())
	return &gzipEntry{
		body:        buf.Bytes(),
		etag:        `"` + hex.EncodeToString(sum[:8]) + `-gz"`,
		contentType: ct,
		ok:          true,
	}
}
