package gin

import (
	"encoding/json"
	"net/http"
)

type HandlerFunc func(*Context)

type H map[string]any

type Engine struct {
	routes map[string]HandlerFunc
}

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
}

func Default() *Engine {
	return New()
}

func New() *Engine {
	return &Engine{routes: make(map[string]HandlerFunc)}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := e.routes[key(r.Method, r.URL.Path)]; ok {
		handler(&Context{Writer: w, Request: r})
		return
	}
	http.NotFound(w, r)
}

func (e *Engine) Use(...HandlerFunc) {}

func Logger() HandlerFunc   { return func(*Context) {} }
func Recovery() HandlerFunc { return func(*Context) {} }

func (e *Engine) addRoute(method, path string, handler HandlerFunc) {
	e.routes[key(method, path)] = handler
}

func (e *Engine) GET(path string, handler HandlerFunc) {
	e.addRoute(http.MethodGet, path, handler)
}

func (c *Context) JSON(code int, obj any) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)
	_ = json.NewEncoder(c.Writer).Encode(obj)
}

func key(method, path string) string {
	return method + " " + path
}
