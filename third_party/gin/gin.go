package gin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

// H is a shortcut map type for building JSON responses.
type H map[string]any

// HandlerFunc defines the handler used by middleware and routes.
type HandlerFunc func(*Context)

// Engine is the central router type, compatible with http.Handler.
type Engine struct {
	routes     []route
	middleware []HandlerFunc
}

type route struct {
	method   string
	pattern  []string
	handlers []HandlerFunc
}

// Context carries the request-specific values passed through handlers.
type Context struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	index    int
	handlers []HandlerFunc
	keys     sync.Map
	status   int
	params   map[string]string
}

// Default creates a new Engine instance with default middleware placeholders.
func Default() *Engine {
	return &Engine{}
}

// ServeHTTP satisfies the http.Handler interface.
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathSegments := splitPath(r.URL.Path)
	for _, route := range e.routes {
		if route.method != r.Method {
			continue
		}
		params, ok := matchRoute(route.pattern, pathSegments)
		if !ok {
			continue
		}

		allHandlers := make([]HandlerFunc, 0, len(e.middleware)+len(route.handlers))
		allHandlers = append(allHandlers, e.middleware...)
		allHandlers = append(allHandlers, route.handlers...)

		ctx := &Context{
			Writer:   w,
			Request:  r,
			index:    -1,
			handlers: allHandlers,
			status:   http.StatusOK,
			params:   params,
		}

		ctx.Next()
		return
	}

	http.NotFound(w, r)
}

// Use registers middleware handlers applied to every request.
func (e *Engine) Use(handlers ...HandlerFunc) {
	e.middleware = append(e.middleware, handlers...)
}

// GET registers a new GET route for the provided path.
func (e *Engine) GET(path string, handlers ...HandlerFunc) {
	e.handle(http.MethodGet, path, handlers...)
}

// POST registers a new POST route for the provided path.
func (e *Engine) POST(path string, handlers ...HandlerFunc) {
	e.handle(http.MethodPost, path, handlers...)
}

// PUT registers a new PUT route for the provided path.
func (e *Engine) PUT(path string, handlers ...HandlerFunc) {
	e.handle(http.MethodPut, path, handlers...)
}

// DELETE registers a new DELETE route for the provided path.
func (e *Engine) DELETE(path string, handlers ...HandlerFunc) {
	e.handle(http.MethodDelete, path, handlers...)
}

// handle binds an HTTP method and path to the provided handlers.
func (e *Engine) handle(method, path string, handlers ...HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	e.routes = append(e.routes, route{
		method:   method,
		pattern:  splitPath(path),
		handlers: handlers,
	})
}

// Next advances to the next handler in the chain.
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		handler := c.handlers[c.index]
		handler(c)
		c.index++
	}
}

// Abort prevents remaining handlers from executing.
func (c *Context) Abort() {
	c.index = len(c.handlers)
}

// JSON writes the JSON response with the provided status code.
func (c *Context) JSON(statusCode int, obj any) {
	c.status = statusCode
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(statusCode)
	_ = json.NewEncoder(c.Writer).Encode(obj)
}

// Status sets the response status code without writing a body.
func (c *Context) Status(statusCode int) {
	c.status = statusCode
	c.Writer.WriteHeader(statusCode)
}

// Set stores a key/value pair in the context.
func (c *Context) Set(key string, value any) {
	c.keys.Store(key, value)
}

// Get retrieves a value previously stored with Set.
func (c *Context) Get(key string) (any, bool) {
	return c.keys.Load(key)
}

// AbortWithStatusJSON aborts the handler chain and writes a JSON response.
func (c *Context) AbortWithStatusJSON(statusCode int, obj any) {
	c.Abort()
	c.JSON(statusCode, obj)
}

// StatusCode returns the most recent status code written to the response.
func (c *Context) StatusCode() int {
	return c.status
}

// ShouldBindJSON decodes the request body into the provided struct.
func (c *Context) ShouldBindJSON(obj any) error {
	if c.Request.Body == nil {
		return errors.New("empty body")
	}
	decoder := json.NewDecoder(c.Request.Body)
	return decoder.Decode(obj)
}

// Param retrieves a named parameter from the request path.
func (c *Context) Param(key string) string {
	if c.params == nil {
		return ""
	}
	return c.params[key]
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func matchRoute(pattern, path []string) (map[string]string, bool) {
	if len(pattern) != len(path) {
		return nil, false
	}
	params := make(map[string]string)
	for i := range pattern {
		segment := pattern[i]
		if strings.HasPrefix(segment, ":") {
			params[segment[1:]] = path[i]
			continue
		}
		if segment != path[i] {
			return nil, false
		}
	}
	return params, true
}
