package server

import "github.com/evo-cloud/hmake/server/models"

func (s *Server) apiGetEnv(ctx *HandlerCtx) {
	var env models.ServerEnv
	for name := range s.scmNames {
		env.SCMProviders = append(env.SCMProviders, name)
	}
	ctx.JSON(&env)
}

func (s *Server) apiListRepos(ctx *HandlerCtx) {

}

func (s *Server) apiUpdateRepo(ctx *HandlerCtx) {

}

func (s *Server) apiRegisterRepo(ctx *HandlerCtx) {

}

func (s *Server) apiDeregisterRepo(ctx *HandlerCtx) {

}
