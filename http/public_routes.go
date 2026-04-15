package http

import (
	"context"
	"net/http"

	"github.com/openbao/openbao/vault"
)

const (
	PublicRouteRequestContextKey = "public_route_request"
)

func wrapPublicRoutesHandler(h http.Handler, props *vault.HandlerProperties) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if props != nil && props.ListenerConfig != nil && props.ListenerConfig.PublicRoutes {
			ctx = context.WithValue(req.Context(), PublicRouteRequestContextKey, true)
		}

		h.ServeHTTP(writer, req.WithContext(ctx))
	})
}
