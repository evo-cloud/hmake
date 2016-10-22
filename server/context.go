package server

import (
	"context"
	"net/http"
	"net/url"
)

type serverCtxBase struct {
	server        *Server
	logger        Logger
	configSection string
}

func (c *serverCtxBase) GlobalConfig() Config {
	return c.server.Config
}

func (c *serverCtxBase) GetConfig(out interface{}) error {
	return c.server.ConfigManager.Get(c.configSection, out)
}

func (c *serverCtxBase) Log() Logger {
	return c.logger
}

func (c *serverCtxBase) Scheduler() Scheduler {
	return c.server.Scheduler()
}

type moduleInitCtx struct {
	serverCtxBase
	module Module
	logger Logger
}

// NewModuleInitCtx creates init ctx for modules
func NewModuleInitCtx(s *Server, m Module) InitCtx {
	c := &moduleInitCtx{module: m}
	c.server = s
	c.logger = s.Log().With("module", m.Name())
	c.configSection = m.Name()
	return c
}

func (c *moduleInitCtx) AddSCMProvider(name string, provider SCMProvider) error {
	return c.server.AddSCMProvider(c.module, name, provider)
}

func (c *moduleInitCtx) AddHandler(path string, handler http.Handler) (*url.URL, error) {
	return c.server.AddModuleHandler(c.module, path, handler)
}

func (c *moduleInitCtx) AddRequestFilter(filter RequestFilter) error {
	return c.server.AddRequestFilter(c.module, filter)
}

const (
	requestIDHeader = "X-Request-ID"
	requestCtxKey   = "request"
)

type requestCtx struct {
	serverCtxBase
	requestID      string
	request        *http.Request
	responseWriter http.ResponseWriter
	module         Module
}

func newRequestCtx(s *Server, r *http.Request, w http.ResponseWriter) RequestCtx {
	ctx := &requestCtx{
		requestID:      r.Header.Get(requestIDHeader),
		responseWriter: w,
	}
	ctx.server = s
	meta := map[string]interface{}{
		"method": r.Method,
		"url":    r.URL,
		"remote": r.RemoteAddr,
	}
	if ctx.requestID != "" {
		meta["req-id"] = ctx.requestID
	}
	ctx.logger = s.Log().With(meta)
	ctx.request = r.WithContext(context.WithValue(r.Context(), requestCtxKey, ctx))
	return ctx
}

func (c *requestCtx) Request() *http.Request {
	return c.request
}

func (c *requestCtx) ResponseWriter() http.ResponseWriter {
	return c.responseWriter
}

func (c *requestCtx) RequestID() string {
	return c.requestID
}

func (c *requestCtx) Module() Module {
	return c.module
}

func (c *requestCtx) forModule(m Module) {
	c.module = m
	c.logger = c.logger.With("module", m.Name())
	c.configSection = m.Name()
}

// CtxFromRequest extracts RequestCtx from request
func CtxFromRequest(r *http.Request) RequestCtx {
	val := r.Context().Value(requestCtxKey)
	if val == nil {
		panic("RequestCtx not found")
	}
	return val.(*requestCtx)
}

func requestForModule(r *http.Request, m Module) *http.Request {
	CtxFromRequest(r).(*requestCtx).forModule(m)
	return r
}
