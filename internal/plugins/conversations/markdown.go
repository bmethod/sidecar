package conversations

import (
	"log"
	"strings"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/charmbracelet/glamour"
)

const (
	minWidthForMarkdown = 30
	maxCacheEntries     = 100
)

// GlamourRenderer wraps Glamour for markdown rendering with caching.
type GlamourRenderer struct {
	mu        sync.RWMutex
	renderer  *glamour.TermRenderer
	lastWidth int
	cache     map[uint64][]string
}

// NewGlamourRenderer creates a new renderer instance.
func NewGlamourRenderer() (*GlamourRenderer, error) {
	return &GlamourRenderer{
		cache: make(map[uint64][]string),
	}, nil
}

// RenderContent renders markdown content to styled lines.
func (r *GlamourRenderer) RenderContent(content string, width int) []string {
	if width < minWidthForMarkdown {
		return wrapText(content, width)
	}

	if content == "" {
		return []string{}
	}

	key := r.cacheKey(content, width)

	// Check cache first (read lock)
	r.mu.RLock()
	if cached, ok := r.cache[key]; ok {
		r.mu.RUnlock()
		return cached
	}
	r.mu.RUnlock()

	// Need to render (write lock)
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check cache after acquiring write lock
	if cached, ok := r.cache[key]; ok {
		return cached
	}

	// Get or create renderer for this width
	renderer, err := r.getOrCreateRenderer(width)
	if err != nil {
		log.Printf("glamour renderer error: %v", err)
		return wrapText(content, width)
	}

	// Render markdown
	rendered, err := renderer.Render(content)
	if err != nil {
		log.Printf("glamour render error: %v", err)
		return wrapText(content, width)
	}

	// Trim trailing whitespace and split into lines
	rendered = strings.TrimRight(rendered, "\n\r\t ")
	lines := strings.Split(rendered, "\n")

	// Cache eviction if needed
	if len(r.cache) >= maxCacheEntries {
		r.cache = make(map[uint64][]string)
	}
	r.cache[key] = lines

	return lines
}

// cacheKey generates a cache key from content and width using xxhash.
func (r *GlamourRenderer) cacheKey(content string, width int) uint64 {
	h := xxhash.New()
	h.WriteString(content)
	h.Write([]byte{byte(width >> 8), byte(width)})
	return h.Sum64()
}

// getOrCreateRenderer lazily creates or recreates the renderer for the given width.
// Must be called with write lock held.
func (r *GlamourRenderer) getOrCreateRenderer(width int) (*glamour.TermRenderer, error) {
	if r.renderer != nil && r.lastWidth == width {
		return r.renderer, nil
	}

	// Width changed or first use - create new renderer and clear cache
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	r.renderer = renderer
	r.lastWidth = width
	r.cache = make(map[uint64][]string) // Clear cache on width change

	return renderer, nil
}
