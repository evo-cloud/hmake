package server

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Server is the server instance
type Server struct {
	Config        Config
	ConfigManager ConfigManager
	Logger        Logger

	scmProviders []*registeredProvider
	scmNames     map[string]*registeredProvider

	reqFilters []RequestFilter

	baseURL *url.URL
	mux     *http.ServeMux
}

type registeredProvider struct {
	module   Module
	provider SCMProvider
}

const (
	apiBase       = "/v1/"
	moduleAPIBase = apiBase + "m/"
)

// NewServer creates a server instance
func NewServer(cfgmgr ConfigManager) (*Server, error) {
	s := &Server{
		ConfigManager: cfgmgr,
		scmNames:      make(map[string]*registeredProvider),
	}
	err := cfgmgr.Get("server", &s.Config)
	if err != nil {
		return nil, err
	}
	if s.Config.BaseURL == "" {
		return nil, MissingConfig("base-url")
	}
	if s.baseURL, err = url.Parse(s.Config.BaseURL); err != nil {
		return nil, NewErrorMsgCause("bad base-url", err)
	}

	s.Logger, err = NewLogger(cfgmgr)
	if err != nil {
		return nil, err
	}

	s.mux = http.NewServeMux()
	s.mux.Handle(apiBase+"env", HandlerFunc(s.handleEnv))
	s.mux.Handle(apiBase+"repos/", HandlerFunc(s.handleRepos))
	s.mux.Handle("/lib/", http.StripPrefix("/lib/", http.FileServer(http.Dir("web/bower_components"))))
	s.mux.Handle("/", http.FileServer(http.Dir("web/www")))
	return s, nil
}

// Serve starts the server
func (s *Server) Serve() error {
	modules := Modules()
	for _, m := range modules {
		ctx := NewModuleInitCtx(s, m)
		ctx.Log().Info("init")
		err := m.Init(ctx)
		ctx.Log().With(err).Error("init failure")
	}
	addr := fmt.Sprintf("%s:%d", s.Config.BindAddress, s.Config.Port)
	handler := http.HandlerFunc(s.preHandler)
	if s.Config.TLSCertFile != "" {
		if s.Config.TLSKeyFile == "" {
			return MissingConfig("tls-key-file")
		}
		return http.ListenAndServeTLS(addr,
			s.Config.TLSCertFile, s.Config.TLSKeyFile, handler)
	}
	return http.ListenAndServe(addr, handler)
}

// Log retrieve the logger
func (s *Server) Log() Logger {
	return s.Logger
}

// Scheduler returns the scheduler
func (s *Server) Scheduler() Scheduler {
	return s
}

// AddSCMProvider registers a source management provider
// Returns os.ErrExist if the name already registered
func (s *Server) AddSCMProvider(module Module, name string, provider SCMProvider) error {
	if _, exist := s.scmNames[name]; exist {
		return os.ErrExist
	}
	prov := &registeredProvider{module: module, provider: provider}
	s.scmProviders = append(s.scmProviders, prov)
	s.scmNames[name] = prov
	return nil
}

// AddModuleHandler registers http.Handler under module's namespace
func (s *Server) AddModuleHandler(module Module, path string, handler http.Handler) (fullURL *url.URL, err error) {
	urlPath := moduleAPIBase + module.Name() + "/"
	if path = strings.TrimLeft(path, "/"); path != "" {
		urlPath += path
	}
	if fullURL, err = url.Parse(s.baseURL.String() + urlPath); err == nil {
		s.mux.Handle(urlPath, s.moduleHandler(module, handler))
	}
	return
}

// AddRequestFilter registers a filter pre-process requests
func (s *Server) AddRequestFilter(module Module, filter RequestFilter) error {
	s.reqFilters = append(s.reqFilters, filter)
	return nil
}

func (s *Server) moduleHandler(module Module, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, requestForModule(r, module))
	})
}

type requestFilterRunner struct {
	filters []RequestFilter
	current int
	handler http.Handler
}

func (f *requestFilterRunner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if f.current < len(f.filters) {
		filter := f.filters[f.current]
		f.current++
		filter(w, r, f)
	} else {
		f.handler.ServeHTTP(w, r)
	}
}

func (s *Server) preHandler(w http.ResponseWriter, r *http.Request) {
	w = s.traceRequest(w, r)
	ctx := newRequestCtx(s, r, w)
	runner := &requestFilterRunner{filters: s.reqFilters, handler: s.mux}
	runner.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
}

func (s *Server) handleEnv(ctx *HandlerCtx) {
	ctx.ByMethod().
		Get(HandlerFunc(s.apiGetEnv)).
		Reject()
}

func (s *Server) handleRepos(ctx *HandlerCtx) {
	ctx.ByMethod().
		Get(HandlerFunc(s.apiListRepos)).
		Put(HandlerFunc(s.apiUpdateRepo)).
		Post(HandlerFunc(s.apiRegisterRepo)).
		Delete(HandlerFunc(s.apiDeregisterRepo)).
		Reject()
}
