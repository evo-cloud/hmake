package server

import (
	"encoding/json"
	"net/http"
)

// HandlerCtx helps implement http.Handler
type HandlerCtx struct {
	w          http.ResponseWriter
	r          *http.Request
	ctx        RequestCtx
	statusCode int
}

// HandlerFunc defines the callback using *HandlerCtx as parameter
type HandlerFunc func(*HandlerCtx)

// ServeHTTP implements http.Handler
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(&HandlerCtx{w: w, r: r, ctx: CtxFromRequest(r)})
}

// MethodDispatcher dispatches HTTP handler by method
type MethodDispatcher interface {
	Get(http.Handler) MethodDispatcher
	Put(http.Handler) MethodDispatcher
	Post(http.Handler) MethodDispatcher
	Delete(http.Handler) MethodDispatcher
	Otherwise(http.Handler) MethodDispatcher
	Reject() MethodDispatcher
}

// Implement Context

// GlobalConfig implements Context
func (c *HandlerCtx) GlobalConfig() Config {
	return c.ctx.GlobalConfig()
}

// GetConfig implements Context
func (c *HandlerCtx) GetConfig(out interface{}) error {
	return c.ctx.GetConfig(out)
}

// Log implements Context
func (c *HandlerCtx) Log() Logger {
	return c.ctx.Log()
}

// Scheduler implements Context
func (c *HandlerCtx) Scheduler() Scheduler {
	return c.ctx.Scheduler()
}

// Implement RequestCtx

// Request implements RequestCtx
func (c *HandlerCtx) Request() *http.Request {
	return c.r
}

// ResponseWriter implements RequestCtx
func (c *HandlerCtx) ResponseWriter() http.ResponseWriter {
	return c.w
}

// RequestID implements RequestCtx
func (c *HandlerCtx) RequestID() string {
	return c.ctx.RequestID()
}

// Module implements RequestCtx
func (c *HandlerCtx) Module() Module {
	return c.ctx.Module()
}

// Helper functions

type methodDispatcher struct {
	ctx        *HandlerCtx
	dispatched bool
}

// Get implements MethodDispatcher
func (d *methodDispatcher) Get(handler http.Handler) MethodDispatcher {
	if d.ctx.Request().Method == http.MethodGet {
		d.ctx.Serve(handler)
		d.dispatched = true
	}
	return d
}

// Put implements MethodDispatcher
func (d *methodDispatcher) Put(handler http.Handler) MethodDispatcher {
	if d.ctx.Request().Method == http.MethodPut {
		d.ctx.Serve(handler)
		d.dispatched = true
	}
	return d
}

// Post implements MethodDispatcher
func (d *methodDispatcher) Post(handler http.Handler) MethodDispatcher {
	if d.ctx.Request().Method == http.MethodPost {
		d.ctx.Serve(handler)
		d.dispatched = true
	}
	return d
}

// Delete implements MethodDispatcher
func (d *methodDispatcher) Delete(handler http.Handler) MethodDispatcher {
	if d.ctx.Request().Method == http.MethodDelete {
		d.ctx.Serve(handler)
		d.dispatched = true
	}
	return d
}

func (d *methodDispatcher) Otherwise(handler http.Handler) MethodDispatcher {
	if !d.dispatched {
		d.ctx.Serve(handler)
		d.dispatched = true
	}
	return d
}

func (d *methodDispatcher) Reject() MethodDispatcher {
	if !d.dispatched {
		d.ctx.MethodNotAllowed(nil)
	}
	return d
}

// ByMethod returns method dispatcher
func (c *HandlerCtx) ByMethod() MethodDispatcher {
	return &methodDispatcher{ctx: c}
}

// Serve calls http.Handler to handle current request
func (c *HandlerCtx) Serve(handler http.Handler) {
	handler.ServeHTTP(c.ResponseWriter(), c.Request())
}

// Redirect wraps http.Redirect
func (c *HandlerCtx) Redirect(url string) {
	http.Redirect(c.w, c.r, url, c.codeIfNotSet(http.StatusTemporaryRedirect))
}

// Status sets the HTTP status code
func (c *HandlerCtx) Status(code int) *HandlerCtx {
	c.statusCode = code
	return c
}

// JSON writes a JSON response
func (c *HandlerCtx) JSON(v interface{}) {
	if v == nil {
		c.NoContent()
		return
	}
	code := c.statusCode
	if _, ok := v.(error); ok && code == 0 {
		code = http.StatusInternalServerError
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	c.w.Header().Add("Content-type", "application/json")
	if code != 0 {
		c.w.WriteHeader(code)
	}
	c.w.Write(encoded)
}

// NoContent is a shortcut to write no-content response
func (c *HandlerCtx) NoContent() {
	c.w.WriteHeader(c.codeIfNotSet(http.StatusNoContent))
	c.w.Write(nil)
}

// BadRequest is a shortcut for write bad request response
func (c *HandlerCtx) BadRequest(err error) {
	c.Status(http.StatusBadRequest).JSON(err)
}

// Unauthorized is a shortcut for write unauthorized response
func (c *HandlerCtx) Unauthorized(err error) {
	c.Status(http.StatusUnauthorized).JSON(err)
}

// Forbidden is a shortcut for write forbidden response
func (c *HandlerCtx) Forbidden(err error) {
	c.Status(http.StatusForbidden).JSON(err)
}

// NotFound is a shortcut to write not found response
func (c *HandlerCtx) NotFound(err error) {
	c.Status(http.StatusNotFound).JSON(err)
}

// MethodNotAllowed is a shortcut to write method not allowed response
func (c *HandlerCtx) MethodNotAllowed(err error) {
	c.Status(http.StatusMethodNotAllowed).JSON(err)
}

// Timeout is a shortcut for write request timeout response
func (c *HandlerCtx) Timeout(err error) {
	c.Status(http.StatusRequestTimeout).JSON(err)
}

// Conflict is a shortcut for write conflict response
func (c *HandlerCtx) Conflict(err error) {
	c.Status(http.StatusConflict).JSON(err)
}

func (c *HandlerCtx) codeIfNotSet(code int) int {
	statusCode := c.statusCode
	if statusCode == 0 {
		statusCode = code
	}
	return statusCode
}
