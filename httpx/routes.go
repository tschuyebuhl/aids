package httpx

import (
	"log/slog"
	"net/http"
)

type Route struct {
	Pattern string // e.g. "POST /api/habits"
	Handler http.HandlerFunc
	Use     []Middleware
}

type Routable interface {
	Routes() []Route
}

type RoutableFunc func() []Route

func (f RoutableFunc) Routes() []Route {
	if f == nil {
		return nil
	}
	return f()
}

type staticRoutes []Route

func (s staticRoutes) Routes() []Route {
	return []Route(s)
}

func Routes(rs ...Route) Routable {
	return staticRoutes(rs)
}

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type routeGroup struct {
	base Routable
	use  []Middleware
}

func (g routeGroup) Routes() []Route {
	if g.base == nil {
		return nil
	}
	routes := g.base.Routes()
	if len(routes) == 0 || len(g.use) == 0 {
		return routes
	}
	out := make([]Route, 0, len(routes))
	for _, rt := range routes {
		use := make([]Middleware, 0, len(g.use)+len(rt.Use))
		use = append(use, g.use...)
		use = append(use, rt.Use...)
		rt.Use = use
		out = append(out, rt)
	}
	return out
}

func With(use ...Middleware) func(Routable) Routable {
	return func(r Routable) Routable {
		return routeGroup{base: r, use: use}
	}
}

func Use(r Routable, use ...Middleware) Routable {
	return routeGroup{base: r, use: use}
}

func Register(mux *http.ServeMux, rs ...Routable) {
	for _, r := range rs {
		for _, rt := range r.Routes() {
			slog.Debug("registering route", "pattern", rt.Pattern)
			h := Chain(http.HandlerFunc(rt.Handler), rt.Use...)
			mux.Handle(rt.Pattern, h)
		}
	}
}
