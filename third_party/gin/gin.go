package gin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// HandlerFunc defines the handler used by gin.
type HandlerFunc func(*Context)

// H is a shortcut for creating JSON responses.
type H map[string]any

// Engine stores registered routes and middleware.
type Engine struct {
	routes      []*route
	middlewares []HandlerFunc
	pool        sync.Pool
}

// RouterGroup represents a route grouping with shared middleware and prefix.
type RouterGroup struct {
	engine   *Engine
	basePath string
	handlers []HandlerFunc
}

type route struct {
	method   string
	path     string
	handlers []HandlerFunc
	parts    []pathPart
}

type pathPart struct {
	value string
	param bool
}

// Context carries request-scoped values and helpers.
type Context struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	handlers []HandlerFunc
	index    int
	params   map[string]string
	keys     map[string]any
	status   int
	aborted  bool
}

// Default returns a new Engine with no-op logger and recovery middleware wired.
func Default() *Engine {
	e := New()
	e.Use(Logger(), Recovery())
	return e
}

// New creates an Engine instance.
func New() *Engine {
	e := &Engine{}
	e.pool.New = func() any { return &Context{} }
	return e
}

// ServeHTTP matches the incoming request against the registered routes.
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rt := range e.routes {
		if rt.method != r.Method {
			continue
		}
		params, ok := matchPath(rt.parts, r.URL.Path)
		if !ok {
			continue
		}
		ctx := e.getContext(w, r, params, rt.handlers)
		ctx.Next()
		e.putContext(ctx)
		return
	}
	http.NotFound(w, r)
}

// Use appends middleware handlers to the engine.
func (e *Engine) Use(handlers ...HandlerFunc) {
	e.middlewares = append(e.middlewares, handlers...)
}

// Logger provides a placeholder logger middleware.
func Logger() HandlerFunc { return func(*Context) {} }

// Recovery provides a placeholder recovery middleware.
func Recovery() HandlerFunc { return func(*Context) {} }

func (e *Engine) addRoute(method, path string, handlers []HandlerFunc) {
	parts := parsePath(path)
	combined := append([]HandlerFunc{}, e.middlewares...)
	combined = append(combined, handlers...)
	e.routes = append(e.routes, &route{method: method, path: path, handlers: combined, parts: parts})
}

// Group creates a new router group beneath the engine.
func (e *Engine) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		engine:   e,
		basePath: joinPaths("", relativePath),
		handlers: handlers,
	}
}

func joinPaths(base, relative string) string {
	if base == "" {
		base = "/"
	}
	if relative == "" {
		return base
	}
	if strings.HasSuffix(base, "/") {
		base = strings.TrimSuffix(base, "/")
	}
	if !strings.HasPrefix(relative, "/") {
		relative = "/" + relative
	}
	return base + relative
}

func (g *RouterGroup) handle(method, relativePath string, handlers []HandlerFunc) {
	if g == nil || g.engine == nil {
		return
	}
	path := joinPaths(g.basePath, relativePath)
	chain := append([]HandlerFunc{}, g.handlers...)
	chain = append(chain, handlers...)
	g.engine.addRoute(method, path, chain)
}

// Use appends middleware to the group.
func (g *RouterGroup) Use(handlers ...HandlerFunc) {
	g.handlers = append(g.handlers, handlers...)
}

func (g *RouterGroup) GET(relativePath string, handlers ...HandlerFunc) {
	g.handle(http.MethodGet, relativePath, handlers)
}

func (g *RouterGroup) POST(relativePath string, handlers ...HandlerFunc) {
	g.handle(http.MethodPost, relativePath, handlers)
}

func (g *RouterGroup) PUT(relativePath string, handlers ...HandlerFunc) {
	g.handle(http.MethodPut, relativePath, handlers)
}

func (g *RouterGroup) DELETE(relativePath string, handlers ...HandlerFunc) {
	g.handle(http.MethodDelete, relativePath, handlers)
}

// HTTP verb helpers ---------------------------------------------------------

func (e *Engine) GET(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodGet, path, handlers)
}

func (e *Engine) POST(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodPost, path, handlers)
}

func (e *Engine) PUT(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodPut, path, handlers)
}

func (e *Engine) DELETE(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodDelete, path, handlers)
}

func (e *Engine) PATCH(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodPatch, path, handlers)
}

func (e *Engine) OPTIONS(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodOptions, path, handlers)
}

// Context helpers -----------------------------------------------------------

// Next advances to the next handler in the chain.
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		handler := c.handlers[c.index]
		handler(c)
		c.index++
		if c.aborted {
			return
		}
	}
}

// Abort prevents remaining handlers from executing.
func (c *Context) Abort() {
	c.aborted = true
	c.index = len(c.handlers)
}

// AbortWithStatusJSON aborts the chain and writes a JSON response.
func (c *Context) AbortWithStatusJSON(code int, obj any) {
	c.aborted = true
	c.JSON(code, obj)
}

// Status sets the HTTP status code on the response writer.
func (c *Context) Status(code int) {
	c.status = code
	if rw, ok := c.Writer.(*responseWriter); ok {
		rw.WriteHeader(code)
		return
	}
	c.Writer.WriteHeader(code)
}

// JSON writes a JSON response with the supplied status code.
func (c *Context) JSON(code int, obj any) {
	if code == 0 {
		code = http.StatusOK
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Status(code)
	enc := json.NewEncoder(c.Writer)
	_ = enc.Encode(obj)
}

// ShouldBindJSON unmarshals the request body into obj.
func (c *Context) ShouldBindJSON(obj any) error {
	if c.Request == nil || c.Request.Body == nil {
		return io.EOF
	}
	decoder := json.NewDecoder(c.Request.Body)
	return decoder.Decode(obj)
}

// BindJSON mirrors gin's BindJSON alias.
func (c *Context) BindJSON(obj any) error {
	return c.ShouldBindJSON(obj)
}

// Param retrieves a path parameter.
func (c *Context) Param(key string) string {
	if c.params == nil {
		return ""
	}
	return c.params[key]
}

// Query fetches a query parameter.
func (c *Context) Query(key string) string {
	if c.Request == nil {
		return ""
	}
	return c.Request.URL.Query().Get(key)
}

// GetHeader retrieves a header value from the incoming request.
func (c *Context) GetHeader(key string) string {
	if c.Request == nil {
		return ""
	}
	return c.Request.Header.Get(key)
}

// DefaultQuery returns the value for key or def if not present.
func (c *Context) DefaultQuery(key, def string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	return def
}

// Set stores a value in the context.
func (c *Context) Set(key string, value any) {
	if c.keys == nil {
		c.keys = make(map[string]any)
	}
	c.keys[key] = value
}

// Get returns a value stored via Set.
func (c *Context) Get(key string) (any, bool) {
	if c.keys == nil {
		return nil, false
	}
	v, ok := c.keys[key]
	return v, ok
}

// MustGet returns the stored value or panics.
func (c *Context) MustGet(key string) any {
	if v, ok := c.Get(key); ok {
		return v
	}
	panic("gin: key not set")
}

// Helper functions ----------------------------------------------------------

func (e *Engine) getContext(w http.ResponseWriter, r *http.Request, params map[string]string, handlers []HandlerFunc) *Context {
	ctx := e.pool.Get().(*Context)
	ctx.Writer = &responseWriter{ResponseWriter: w}
	ctx.Request = r
	ctx.params = params
	ctx.handlers = handlers
	ctx.index = -1
	ctx.keys = nil
	ctx.status = 0
	ctx.aborted = false
	return ctx
}

func (e *Engine) putContext(ctx *Context) {
	ctx.Writer = nil
	ctx.Request = nil
	ctx.handlers = nil
	ctx.params = nil
	ctx.keys = nil
	e.pool.Put(ctx)
}

type responseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		if !w.wroteHeader {
			w.WriteHeader(http.StatusOK)
		}
		f.Flush()
	}
}

func matchPath(parts []pathPart, rawPath string) (map[string]string, bool) {
	path := rawPath
	if path == "" {
		path = "/"
	}
	incoming := splitPath(path)
	if len(incoming) != len(parts) {
		return nil, false
	}
	params := make(map[string]string)
	for i, part := range parts {
		seg := incoming[i]
		if part.param {
			params[part.value] = seg
			continue
		}
		if part.value != seg {
			return nil, false
		}
	}
	return params, true
}

func parsePath(path string) []pathPart {
	segments := splitPath(path)
	parts := make([]pathPart, len(segments))
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			parts[i] = pathPart{value: strings.TrimPrefix(seg, ":"), param: true}
			continue
		}
		parts[i] = pathPart{value: seg}
	}
	return parts
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return []string{""}
	}
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{""}
	}
	parts := strings.Split(trimmed, "/")
	for i, p := range parts {
		if decoded, err := url.PathUnescape(p); err == nil {
			parts[i] = decoded
		}
	}
	return parts
}
