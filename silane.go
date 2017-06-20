package silane

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

/*
##############################
	Silane ServeMux
##############################
*/
type ServeMux struct {
	Map
	NotFound   HandlerFunc
	NotAllowed HandlerFunc
}

func New() *ServeMux {
	return new(ServeMux)
}
func (s *ServeMux) Group(path string, h func(m *Map)) *Map {
	if s.root == nil {
		s.root = new(node)
	}
	segs := strings.Split(path, "/")
	n := s.root
	for _, seg := range segs {
		if n.nodes == nil {
			n.nodes = make(map[string]*node, 0)
		}
		n2, ok := n.nodes[seg]
		if !ok {
			n2 = new(node)
			n.nodes[seg] = n2
		}
		n = n2
	}
	m := &Map{root: n}
	h(m)
	return m
}
func (s *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := &Context{
		Request:  r,
		Response: &Response{Header: new(Header)},
	}
	defer func() {
		if c.error != nil {
			if c.Response.code == 404 && s.NotFound != nil {
				s.NotFound(c)
			} else if c.Response.code == 405 && s.NotAllowed != nil {
				s.NotAllowed(c)
			} else if c.Request.Header.Get("Accept") == "application/json" {
				c.Response.Header.Set("Content-Type", "application/json")
				c.Response.Json(struct {
					Code  int    `json:"code"`
					Error string `json:"error"`
				}{
					Code:  c.error.code,
					Error: c.error.error,
				})
			} else {
				c.Response.Header.Set("Content-Type", "text/plain")
				c.Response.Text(fmt.Sprintf("code:%d\nerror:%s", c.error.code, c.error.error))
			}
		}
		c.Response.write(w)
	}()
	if s.root == nil {
		c.Response.Status(404)
		c.Error("route not registered.", 100)
		return
	}
	segs := strings.Split(r.URL.Path, "/")
	n := s.root
	p := make([]string, 0)
	m := make([]middleware, 0)
	if n.middlewares != nil && len(n.middlewares) > 0 {
		m = append(m, n.middlewares...)
	}
	for _, seg := range segs {
		n2, ok := n.nodes[seg]
		if !ok || n2 == nil {
			n2, ok = n.nodes["*"]
			if !ok || n2 == nil {
				c.Response.Status(404)
				c.Error("route not matched.", 101)
				return
			}
			p = append(p, seg)
		}
		n = n2
		if n.middlewares != nil && len(n.middlewares) > 0 {
			m = append(m, n.middlewares...)
		}
	}
	hi, ok := n.handlers[r.Method]
	if !ok || hi == nil || hi.handler == nil {
		c.Response.Status(405)
		c.Error("method not allowed.", 102)
		return
	}
	if hi.middlewares != nil && len(hi.middlewares) > 0 {
		m = append(m, hi.middlewares...)
	}
	c.Params = make(map[string]string, len(hi.params))
	for i, name := range hi.params {
		c.Params[name] = p[i]
	}
	pipe := &pipeline{stacks: m}
	pipe.Pipe(func(c *Context, next NextFunc) {
		hi.handler(c)
		next(c)
	})
	c.Response.Status(200)
	c.Response.Header.Set("Content-Type", "text/plain")
	pipe.Run(c)
	if c.error != nil {
		c.Response.Status(403)
	}
}

/*
##############################
	Route Node
##############################
*/
type node struct {
	nodes       map[string]*node
	handlers    map[string]*Route
	middlewares []middleware
}

/*
##############################
	Route Group
##############################
*/
type Map struct {
	root *node
}

func (g *Map) Get(path string, handler HandlerFunc) *Route {
	return g.add("GET", path, handler)
}
func (g *Map) Post(path string, handler HandlerFunc) *Route {
	return g.add("POST", path, handler)
}
func (g *Map) Put(path string, handler HandlerFunc) *Route {
	return g.add("PUT", path, handler)
}
func (g *Map) Patch(path string, handler HandlerFunc) *Route {
	return g.add("PATCH", path, handler)
}
func (g *Map) Delete(path string, handler HandlerFunc) *Route {
	return g.add("DELETE", path, handler)
}
func (g *Map) Head(path string, handler HandlerFunc) *Route {
	return g.add("HEAD", path, handler)
}
func (g *Map) Options(path string, handler HandlerFunc) *Route {
	return g.add("OPTIONS", path, handler)
}
func (g *Map) add(method string, path string, handler HandlerFunc) *Route {
	if g.root == nil {
		g.root = new(node)
	}
	segs := strings.Split(path, "/")
	p := make([]string, 0)
	n := g.root
	for _, seg := range segs {
		if len(seg) > 0 && seg[0] == 58 {
			p = append(p, seg[1:])
			seg = "*"
		}
		if n.nodes == nil {
			n.nodes = make(map[string]*node, 0)
		}
		n2, ok := n.nodes[seg]
		if !ok {
			n2 = new(node)
			n.nodes[seg] = n2
		}
		n = n2
	}
	if n.handlers == nil {
		n.handlers = make(map[string]*Route, 0)
	}
	r := &Route{
		handler: handler,
		params:  p,
	}
	n.handlers[method] = r
	return r
}
func (g *Map) Use(handler middleware) {
	if g.root == nil {
		g.root = new(node)
	}
	if g.root.middlewares == nil {
		g.root.middlewares = make([]middleware, 0)
	}
	g.root.middlewares = append(g.root.middlewares, handler)
}

/*
##############################
	Server Context
##############################
*/
type Context struct {
	Request  *http.Request
	Response *Response
	Params   map[string]string
	error    *Error
}

func (c *Context) Error(error string, code int) {
	c.error = &Error{
		code:  code,
		error: error,
	}
}
func (c *Context) GetError() *Error {
	return c.error
}

/*
##############################
	Server Response
##############################
*/
type Response struct {
	Header *Header
	code   int
	body   []byte
}

func (r *Response) Status(code int) {
	r.code = code
}
func (r *Response) Text(text string) {
	r.body = []byte(text)
}
func (r *Response) Json(data interface{}) {
	r.body, _ = json.Marshal(data)
}
func (r *Response) write(w http.ResponseWriter) {
	if r.Header != nil && r.Header.headers != nil {
		for n, v := range r.Header.headers {
			w.Header().Set(n, v)
		}
	}
	if r.body != nil {
		w.Header().Set("Content-Length", strconv.Itoa(len(r.body)))
	}
	if r.code >= 100 {
		w.WriteHeader(r.code)
	} else {
		w.WriteHeader(200)
	}
	if r.body != nil {
		w.Write(r.body)
	}
}

/*
##############################
	Server Response Header
##############################
*/
type Header struct {
	headers map[string]string
}

func (h *Header) Get(name string) (string, bool) {
	if h.headers == nil {
		return "", false
	}
	v, ok := h.headers[name]
	return v, ok
}
func (h *Header) Set(name string, value string) {
	if h.headers == nil {
		h.headers = make(map[string]string)
	}
	h.headers[name] = value
}
func (h *Header) Add(name string, value string) {
	if h.headers == nil {
		h.headers = make(map[string]string)
	}
	v, ok := h.headers[name]
	if ok {
		h.headers[name] = v + "," + value
	} else {
		h.headers[name] = value
	}
}
func (h *Header) Delete(name string) {
	if h.headers == nil {
		return
	}
	_, ok := h.headers[name]
	if ok {
		delete(h.headers, name)
	}
}

/*
##############################
	Handler Holder
##############################
*/
type Route struct {
	handler     HandlerFunc
	params      []string
	middlewares []middleware
}

func (r *Route) With(handler middleware) *Route {
	r.middlewares = append(r.middlewares, handler)
	return r
}

/*
##############################
	Middleware Stream
##############################
*/
type pipeline struct {
	stacks  []middleware
	current int
}

func (m *pipeline) Pipe(h middleware) *pipeline {
	m.stacks = append(m.stacks, h)
	return m
}
func (m *pipeline) Run(c *Context) {
	m.current = 0
	m.next(c)
}
func (m *pipeline) next(c *Context) {
	i := m.current
	m.current += 1
	if i < len(m.stacks) {
		m.stacks[i](c, m.next)
	}
}

/*
##############################
	Error
##############################
*/
type Error struct {
	code  int
	error string
}

func (e *Error) Code() int {
	return e.code
}
func (e *Error) Error() string {
	return e.error
}

/*
##############################
##############################
*/
type HandlerFunc func(c *Context)
type middleware func(c *Context, next NextFunc)
type NextFunc func(c *Context)
