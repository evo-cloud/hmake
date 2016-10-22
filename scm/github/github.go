package github

import (
	"fmt"

	srv "github.com/evo-cloud/hmake/server"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"
)

const (
	// ModuleName is name of the module
	ModuleName = "github"
)

// Config defines the configuration for this module
type Config struct {
	ClientID     string `json:"client-id"`
	ClientSecret string `json:"client-secret"`
}

// Module implements server.Module
type Module struct {
	oauthConf oauth2.Config
}

// Name implements server.Module/server.SCMProvider
func (m *Module) Name() string {
	return ModuleName
}

// Init implements server.Module
func (m *Module) Init(ctx srv.InitCtx) (err error) {
	var conf Config
	if err = ctx.GetConfig(&conf); err != nil {
		return
	}

	if conf.ClientID == "" {
		ctx.Log().With("missing", "client-id").Warn("Deactivated")
		return
	}
	if conf.ClientSecret == "" {
		ctx.Log().With("missing", "client-secret").Warn("Deactivated")
		return
	}

	m.oauthConf.ClientID = conf.ClientID
	m.oauthConf.ClientSecret = conf.ClientSecret
	m.oauthConf.Scopes = []string{"user:email", "repo", "admin:repo_hook", "read:org"}
	m.oauthConf.Endpoint = oauth2github.Endpoint

	ctx.AddSCMProvider(m.Name(), m)
	ctx.AddHandler("/auth", srv.HandlerFunc(m.handleAuth))
	ctx.AddHandler("/auth/callback", srv.HandlerFunc(m.handleAuthCallback))
	ctx.AddHandler("/hook", srv.HandlerFunc(m.handleEvent))

	return nil
}

func (m *Module) handleAuth(ctx *srv.HandlerCtx) {
	url := m.oauthConf.AuthCodeURL("thisshouldberandom", oauth2.AccessTypeOnline)
	ctx.Redirect(url)
}

func (m *Module) handleAuthCallback(ctx *srv.HandlerCtx) {
	state := ctx.Request().FormValue("state")
	if state != "thisshouldberandom" {
		ctx.Redirect("/")
		return
	}

	code := ctx.Request().FormValue("code")
	token, err := m.oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		fmt.Printf("oauthConf.Exchange() failed with '%s'\n", err)
		ctx.Redirect("/")
		return
	}

	oauthClient := m.oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	user, _, err := client.Users.Get("")
	if err != nil {
		fmt.Printf("client.Users.Get() faled with '%s'\n", err)
		ctx.Redirect("/")
		return
	}
	fmt.Printf("Logged in as GitHub user: %s\n", *user.Login)
	ctx.Redirect("/")
}

func (m *Module) handleEvent(ctx *srv.HandlerCtx) {

}

func init() {
	srv.RegisterModule(&Module{})
}
