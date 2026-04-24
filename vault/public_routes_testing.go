package vault

import (
	"context"

	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestPublicRoutesBackendFactory(_ context.Context, _ *logical.BackendConfig) (logical.Backend, error) {
	publicRoutesBackend := testPublicRoutesBackend()
	return publicRoutesBackend, nil
}

func testPublicRoutesBackend() logical.Backend {
	b := &framework.Backend{
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
				"unauthenticated/private",
				"unauthenticated/public",
			},
		},
		Paths: []*framework.Path{
			pathUnauthenticatedPrivate(),
			pathUnauthenticatedPublic(),
		},
		BackendType: logical.TypeLogical,
	}

	config := &logical.BackendConfig{
		System: &logical.StaticSystemView{},
	}

	b.Setup(context.Background(), config)

	return b
}

func pathUnauthenticatedPrivate() *framework.Path {
	return &framework.Path{
		Pattern: "unauthenticated/private",

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: genericHandler,
			},
		},
	}
}

func pathUnauthenticatedPublic() *framework.Path {
	return &framework.Path{
		Pattern: "unauthenticated/public",

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: genericHandler,
			},
		},
	}
}

func genericHandler(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	return &logical.Response{
		Data: map[string]interface{}{
			"message": "ok",
		},
	}, nil
}
