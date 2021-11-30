package api

import (
	"hypefast-api/bootstrap"
	"hypefast-api/services/api/handler"

	"github.com/go-chi/chi"
)

// RegisterRoutes all routes for the apps
func RegisterRoutes(r *chi.Mux, app *bootstrap.App) {
	r.Route("/v1", func(r chi.Router) {
		r.Get("/ping", app.PingAction)

		RegisterSubsRoute(r, app)
	})
}

// RegisterSubsRoute ...
func RegisterSubsRoute(r chi.Router, app *bootstrap.App) chi.Router {
	h := handler.Contract{App: app}
	r.Route("/api", func(r chi.Router) {

		r.Get("/", h.Test)
	})

	return r
}
