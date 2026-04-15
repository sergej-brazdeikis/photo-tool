package share

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"

	webshare "photo-tool/web/share"

	"photo-tool/internal/store"
)

// NotFoundBody is the generic 404 payload for any share-route miss (Story 3.2 AC3).
var NotFoundBody = []byte("Not Found\n")

// ShareHTMLContentSecurityPolicy is the CSP for successful GET/HEAD /s/{token} responses.
// Tight policy: no scripts, no network except same-origin images (/i/{token}), inline styles only (Story 3.4 dev session 2/2).
const ShareHTMLContentSecurityPolicy = "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'; img-src 'self'; style-src 'unsafe-inline'; script-src 'none'"

type handler struct {
	db          *sql.DB
	libraryRoot string
	pageTmpl    *template.Template
	packageTmpl *template.Template
}

// newShareMuxHandler is the inner HTTP handler before rate limiting (Stories 3.2–3.4).
func newShareMuxHandler(db *sql.DB, libraryRoot string) http.Handler {
	tSingle := template.Must(template.New("share.html").ParseFS(webshare.FS, "share.html"))
	tPkg := template.Must(template.New("share_package.html").ParseFS(webshare.FS, "share_package.html"))
	return handler{
		db:          db,
		libraryRoot: libraryRoot,
		pageTmpl:    tSingle,
		packageTmpl: tPkg,
	}
}

// NewHTTPHandler serves GET/HEAD /s/{token} (HTML) and /i/{token} (image bytes) with uniform 404s (Stories 3.2–3.3)
// and a per-client-IP in-memory rate limit (Story 3.5 NFR-06).
func NewHTTPHandler(db *sql.DB, libraryRoot string) http.Handler {
	return wrapRateLimitedHandler(newShareMuxHandler(db, libraryRoot), defaultShareRateLimit())
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := pathpkg.Clean(r.URL.Path)
	if strings.HasPrefix(p, "/s/") {
		token, ok := pathTokenAfterPrefix(p, "/s/")
		h.serveHTML(w, r, token, ok)
		return
	}
	if strings.HasPrefix(p, "/i/") {
		token, memberPos, pkgMember, ok := parseShareImagePath(p)
		h.serveImage(w, r, token, memberPos, pkgMember, ok)
		return
	}
	writeNotFound(w, r)
}

// parseShareImagePath accepts /i/{token} (single) or /i/{token}/{position} (package member). Extra path segments are invalid.
func parseShareImagePath(cleanPath string) (token string, memberPos int, packageMember bool, ok bool) {
	if cleanPath == "" || !strings.HasPrefix(cleanPath, "/i/") {
		return "", 0, false, false
	}
	rest := strings.TrimPrefix(cleanPath, "/i/")
	if rest == "" {
		return "", 0, false, false
	}
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return rest, 0, false, true
	}
	tok := rest[:slash]
	tail := rest[slash+1:]
	if tok == "" || strings.Contains(tail, "/") {
		return "", 0, false, false
	}
	pos, err := strconv.Atoi(tail)
	if err != nil || pos < 0 {
		return "", 0, false, false
	}
	return tok, pos, true, true
}

func pathTokenAfterPrefix(cleanPath, prefix string) (token string, ok bool) {
	if cleanPath == "" || prefix == "" {
		return "", false
	}
	if !strings.HasPrefix(cleanPath, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(cleanPath, prefix)
	if rest == "" || strings.Contains(rest, "/") {
		return "", false
	}
	return rest, true
}

type sharePageTmplData struct {
	InlineCSS   template.CSS
	ImagePath   string
	StarsHTML   template.HTML
	RatingLabel string
}

func (h handler) serveHTML(w http.ResponseWriter, r *http.Request, token string, pathOK bool) {
	if !pathOK {
		writeNotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
	default:
		writeNotFound(w, r)
		return
	}

	pkg, err := store.ResolvePackageShareLink(r.Context(), h.db, token)
	if err != nil {
		slog.Error("share http resolve package", "err", err)
		writeNotFound(w, r)
		return
	}
	if pkg != nil {
		h.servePackageHTML(w, r, token, pkg)
		return
	}

	resolved, err := store.ResolveDefaultShareLink(r.Context(), h.db, token)
	if err != nil {
		slog.Error("share http resolve", "err", err)
		writeNotFound(w, r)
		return
	}
	if resolved == nil {
		writeNotFound(w, r)
		return
	}

	payload, err := store.ParseShareSnapshotPayloadJSON(resolved.Payload)
	if err != nil {
		slog.Error("share http payload", "err", err)
		writeNotFound(w, r)
		return
	}

	cssBytes, err := webshare.FS.ReadFile("share.css")
	if err != nil {
		slog.Error("share http template css", "err", err)
		writeNotFound(w, r)
		return
	}

	stars, label := ratingViewModel(payload)
	data := sharePageTmplData{
		InlineCSS:   template.CSS(cssBytes),
		ImagePath:   "/i/" + token,
		StarsHTML:   stars,
		RatingLabel: label,
	}

	var buf bytes.Buffer
	if err := h.pageTmpl.Execute(&buf, data); err != nil {
		slog.Error("share http render", "err", err)
		writeNotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", ShareHTMLContentSecurityPolicy)
	// Reduce accidental cross-origin referrer leakage on the share document (Story 3.4).
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

type sharePackagePageItem struct {
	ImagePath string
	Alt       string
	Caption   string
}

type sharePackagePageTmplData struct {
	InlineCSS template.CSS
	PageTitle string
	Heading   string
	Summary   string
	Items     []sharePackagePageItem
}

func (h handler) servePackageHTML(w http.ResponseWriter, r *http.Request, rawToken string, pkg *store.ResolvedPackageShareLink) {
	payload, err := store.ParseShareSnapshotPayloadJSON(pkg.Payload)
	if err != nil {
		slog.Error("share http package payload", "err", err)
		writeNotFound(w, r)
		return
	}
	cssBytes, err := webshare.FS.ReadFile("share.css")
	if err != nil {
		slog.Error("share http template css", "err", err)
		writeNotFound(w, r)
		return
	}

	heading := strings.TrimSpace(payload.DisplayTitle)
	if heading == "" {
		heading = "Shared package"
	}
	pageTitle := heading
	summary := fmt.Sprintf("%d photos — shared snapshot", len(pkg.MemberIDs))
	if strings.TrimSpace(payload.AudienceLabel) != "" {
		summary += " · " + strings.TrimSpace(payload.AudienceLabel)
	}

	n := len(pkg.MemberIDs)
	items := make([]sharePackagePageItem, 0, n)
	for i := range pkg.MemberIDs {
		items = append(items, sharePackagePageItem{
			ImagePath: SharePackageMemberImagePath(rawToken, i),
			Alt:       "Shared photo",
			Caption:   fmt.Sprintf("Photo %d of %d", i+1, n),
		})
	}

	data := sharePackagePageTmplData{
		InlineCSS: template.CSS(cssBytes),
		PageTitle: pageTitle,
		Heading:   heading,
		Summary:   summary,
		Items:     items,
	}
	var buf bytes.Buffer
	if err := h.packageTmpl.Execute(&buf, data); err != nil {
		slog.Error("share http package render", "err", err)
		writeNotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", ShareHTMLContentSecurityPolicy)
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

func ratingViewModel(p store.ShareSnapshotPayload) (stars template.HTML, label string) {
	r := 0
	has := false
	if p.Rating != nil {
		r = *p.Rating
		has = true
	}
	if !has || r < 1 || r > 5 {
		return unratedStarsHTML(), "Unrated"
	}
	return ratedStarsHTML(r), "Rating: " + strconv.Itoa(r)
}

const starEmpty = "\u2606"
const starFilled = "\u2605"

func unratedStarsHTML() template.HTML {
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteString(`<span class="star" aria-hidden="true">`)
		b.WriteString(starEmpty)
		b.WriteString(`</span>`)
	}
	return template.HTML(b.String())
}

func ratedStarsHTML(filled int) template.HTML {
	if filled < 1 {
		filled = 1
	}
	if filled > 5 {
		filled = 5
	}
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteString(`<span class="`)
		if i < filled {
			b.WriteString(`star filled`)
		} else {
			b.WriteString(`star`)
		}
		b.WriteString(`" aria-hidden="true">`)
		if i < filled {
			b.WriteString(starFilled)
		} else {
			b.WriteString(starEmpty)
		}
		b.WriteString(`</span>`)
	}
	return template.HTML(b.String())
}

func (h handler) serveImage(w http.ResponseWriter, r *http.Request, token string, memberPos int, packageMember bool, pathOK bool) {
	if !pathOK {
		writeNotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
	default:
		writeNotFound(w, r)
		return
	}

	var assetID int64
	if packageMember {
		pkg, err := store.ResolvePackageShareLink(r.Context(), h.db, token)
		if err != nil {
			slog.Error("share http resolve package image", "err", err)
			writeNotFound(w, r)
			return
		}
		if pkg == nil || memberPos < 0 || memberPos >= len(pkg.MemberIDs) {
			writeNotFound(w, r)
			return
		}
		assetID = pkg.MemberIDs[memberPos]
	} else {
		resolved, err := store.ResolveDefaultShareLink(r.Context(), h.db, token)
		if err != nil {
			slog.Error("share http resolve", "err", err)
			writeNotFound(w, r)
			return
		}
		if resolved == nil {
			writeNotFound(w, r)
			return
		}
		assetID = resolved.AssetID
	}

	relPath, mimeNS, ok, err := store.AssetLibraryFileForShare(r.Context(), h.db, assetID)
	if err != nil {
		slog.Error("share http asset file", "err", err)
		writeNotFound(w, r)
		return
	}
	if !ok {
		writeNotFound(w, r)
		return
	}

	abs, err := store.AssetPrimaryPath(h.libraryRoot, relPath)
	if err != nil {
		writeNotFound(w, r)
		return
	}

	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			writeNotFound(w, r)
			return
		}
		slog.Error("share http image open failed")
		writeNotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()

	st, err := f.Stat()
	if err != nil {
		slog.Error("share http image stat failed")
		writeNotFound(w, r)
		return
	}

	ct := contentTypeForShareImage(mimeNS, f, filepath.Base(abs))
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		slog.Error("share http image seek failed")
		writeNotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-store")
	// Align with HTML route: limit referrer leakage on direct /i/ hits and embeds (Story 3.4).
	w.Header().Set("Referrer-Policy", "no-referrer")
	if strings.HasPrefix(ct, "image/") {
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	http.ServeContent(w, r, filepath.Base(abs), st.ModTime(), f)
}

func contentTypeForShareImage(mime sql.NullString, r io.ReadSeeker, name string) string {
	if mime.Valid {
		m := strings.TrimSpace(mime.String)
		if strings.HasPrefix(strings.ToLower(m), "image/") {
			return m
		}
	}
	hdr := make([]byte, 512)
	n, err := r.Read(hdr)
	if err != nil && err != io.EOF {
		return "application/octet-stream"
	}
	detected := http.DetectContentType(hdr[:n])
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	// Safe fallback: avoid advertising a browser-displayed image type when bytes are not a known image.
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func writeNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// Same Content-Length for GET and HEAD so 404s are not distinguishable by metadata (Story 3.2 AC3).
	w.Header().Set("Content-Length", strconv.Itoa(len(NotFoundBody)))
	if r != nil && r.Method == http.MethodHead {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(NotFoundBody)
}
