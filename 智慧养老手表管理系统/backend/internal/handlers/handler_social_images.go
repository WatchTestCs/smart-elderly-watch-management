package handlers

import (
	"bytes"
	"careold-go/internal/response"
	"encoding/binary"
	"github.com/google/uuid"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math/bits"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImagesApi —— 图片上传下载（PhotoUtil 等价实现，仅标准库）
// ---------------------------------------------------------------------------

const imageFailMessage = "请选择不超过5MB的有效 JPG、PNG 或 BMP 图片"
const maxImageBytes = 5 * 1024 * 1024

// POST /restful/image/saveImage —— 图片上传（multipart/form-data）。
func (h *Handler) imageSave(w http.ResponseWriter, r *http.Request) {
	fail := func() {
		response.JSON(w, http.StatusBadRequest, map[string]any{"returnCode": "0000", "message": imageFailMessage})
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+(1<<20))
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		fail()
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		fail()
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxImageBytes {
		fail()
		return
	}
	format, img, width, height, ok := sniffImage(data)
	if !ok || !validDims(width, height) {
		fail()
		return
	}
	ext := "png"
	if format == "jpeg" {
		ext = "jpg"
	}
	name := uuid.NewString() + "." + ext
	if err := os.MkdirAll(h.Cfg.ImageDir, 0o755); err != nil {
		fail()
		return
	}
	realDir, err := filepath.EvalSymlinks(h.Cfg.ImageDir)
	if err != nil {
		fail()
		return
	}
	tmp, err := os.CreateTemp(realDir, "image-*.tmp")
	if err != nil {
		fail()
		return
	}
	tmpName := tmp.Name()
	encoded := false
	func() {
		defer func() {
			tmp.Close()
			if !encoded {
				os.Remove(tmpName)
			}
		}()
		if ext == "jpg" {
			if err := jpeg.Encode(tmp, img, nil); err != nil {
				return
			}
		} else {
			if err := png.Encode(tmp, img); err != nil {
				return
			}
		}
		encoded = true
	}()
	if !encoded {
		fail()
		return
	}
	// 原子改名落盘（同目录 rename）。
	if err := os.Rename(tmpName, filepath.Join(realDir, name)); err != nil {
		os.Remove(tmpName)
		fail()
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"returnCode": "1111", "filename": name})
}

// sniffImage 按内容嗅探格式并解码像素（jpeg/png 用标准库，bmp 手工解析）。
func sniffImage(data []byte) (format string, img image.Image, width, height int, ok bool) {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil || !validDims(cfg.Width, cfg.Height) {
			return "", nil, 0, 0, false
		}
		m, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return "", nil, 0, 0, false
		}
		return "jpeg", m, cfg.Width, cfg.Height, true
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' &&
		data[4] == 0x0D && data[5] == 0x0A && data[6] == 0x1A && data[7] == 0x0A {
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil || !validDims(cfg.Width, cfg.Height) {
			return "", nil, 0, 0, false
		}
		m, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return "", nil, 0, 0, false
		}
		return "png", m, cfg.Width, cfg.Height, true
	}
	if len(data) >= 2 && data[0] == 'B' && data[1] == 'M' {
		m, w, h, ok := decodeBMP(data)
		if !ok {
			return "", nil, 0, 0, false
		}
		return "bmp", m, w, h, true
	}
	return "", nil, 0, 0, false
}

func validDims(w, h int) bool {
	return w >= 1 && h >= 1 && w <= 8192 && h <= 8192 && int64(w)*int64(h) <= 12000000
}

// decodeBMP 手工解析 BMP（24 位 / 32 位无压缩，32 位 BI_BITFIELDS），返回 RGBA 像素。
func decodeBMP(data []byte) (image.Image, int, int, bool) {
	if len(data) < 54 {
		return nil, 0, 0, false
	}
	off := binary.LittleEndian.Uint32(data[10:14])
	dib := binary.LittleEndian.Uint32(data[14:18])
	if dib < 40 {
		return nil, 0, 0, false
	}
	w := int(int32(binary.LittleEndian.Uint32(data[18:22])))
	rawH := int32(binary.LittleEndian.Uint32(data[22:26]))
	planes := binary.LittleEndian.Uint16(data[26:28])
	bpp := binary.LittleEndian.Uint16(data[28:30])
	comp := binary.LittleEndian.Uint32(data[30:34])
	if planes != 1 || w < 1 || w > 8192 {
		return nil, 0, 0, false
	}
	topDown := false
	h := int(rawH)
	if rawH < 0 {
		h = int(-rawH)
		topDown = true
	}
	if h < 1 || h > 8192 || int64(w)*int64(h) > 12000000 {
		return nil, 0, 0, false
	}
	var rMask, gMask, bMask uint32 = 0x00FF0000, 0x0000FF00, 0x000000FF
	if comp == 3 { // BI_BITFIELDS
		if dib < 52 || len(data) < 66 {
			return nil, 0, 0, false
		}
		rMask = binary.LittleEndian.Uint32(data[54:58])
		gMask = binary.LittleEndian.Uint32(data[58:62])
		bMask = binary.LittleEndian.Uint32(data[62:66])
	} else if comp != 0 {
		return nil, 0, 0, false // 不支持 RLE 等压缩
	}
	if bpp != 24 && bpp != 32 {
		return nil, 0, 0, false
	}
	stride := (w*int(bpp)/8 + 3) &^ 3
	if int64(off)+int64(stride)*int64(h) > int64(len(data)) {
		return nil, 0, 0, false
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := y
		if !topDown {
			sy = h - 1 - y
		}
		row := data[int(off)+sy*stride:]
		for x := 0; x < w; x++ {
			var rr, gg, bb uint8
			if bpp == 24 {
				o := x * 3
				bb, gg, rr = row[o], row[o+1], row[o+2]
			} else {
				px := binary.LittleEndian.Uint32(row[x*4:])
				rr = scaleMask(px, rMask)
				gg = scaleMask(px, gMask)
				bb = scaleMask(px, bMask)
			}
			img.SetRGBA(x, y, color.RGBA{R: rr, G: gg, B: bb, A: 0xFF})
		}
	}
	return img, w, h, true
}

// scaleMask 把掩码抽出的分量缩放到 8 位。
func scaleMask(px, mask uint32) uint8 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	width := bits.OnesCount32(mask)
	v := (px & mask) >> shift
	if width >= 8 {
		return uint8(v >> (width - 8))
	}
	return uint8(v * 255 / ((1 << width) - 1))
}

var imageNameRe = regexp.MustCompile(`(?i)^[^\x00]+\.(png|jpe?g|bmp)$`)

func validImageName(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\:") {
		return false
	}
	return imageNameRe.MatchString(name)
}

// GET /restful/image/download?name=xxx —— 图片下载（多层路径穿越防护）。
func (h *Handler) imageDownload(w http.ResponseWriter, r *http.Request) {
	notFound := func() { w.WriteHeader(http.StatusNotFound) }
	name := r.URL.Query().Get("name")
	if !validImageName(name) {
		notFound()
		return
	}
	fi, err := os.Stat(h.Cfg.ImageDir)
	if err != nil || !fi.IsDir() {
		notFound()
		return
	}
	root, err := filepath.EvalSymlinks(h.Cfg.ImageDir)
	if err != nil {
		notFound()
		return
	}
	candidate := filepath.Join(root, name)
	if rel, err := filepath.Rel(root, candidate); err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		notFound()
		return
	}
	st, err := os.Stat(candidate)
	if err != nil || !st.Mode().IsRegular() {
		notFound()
		return
	}
	real, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		notFound()
		return
	}
	if rel, err := filepath.Rel(root, real); err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		notFound()
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	contentType := "image/jpeg"
	switch ext {
	case "png":
		contentType = "image/png"
	case "bmp":
		contentType = "image/bmp"
	}
	f, err := os.Open(real)
	if err != nil {
		notFound()
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, name, st.ModTime(), f)
}

// ---------------------------------------------------------------------------
