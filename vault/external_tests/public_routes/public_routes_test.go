package public_routes

import (
	"testing"

	"github.com/openbao/openbao/api/v2"
	vaulthttp "github.com/openbao/openbao/http"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/openbao/openbao/vault"
	"github.com/stretchr/testify/require"
)

func TestPublicRoutes_PathAccess(t *testing.T) {
	coreConfig := &vault.CoreConfig{
		LogicalBackends: map[string]logical.Factory{
			"test-backend": vault.TestPublicRoutesBackendFactory,
		},
	}
	cluster := vault.NewTestCluster(t, coreConfig, &vault.TestClusterOptions{
		HandlerFunc:         vaulthttp.Handler,
		PublicRouteListener: true,
		NumCores:            1,
	})
	cluster.Start()
	defer cluster.Cleanup()

	vault.TestWaitActive(t, cluster.Cores[0].Core)
	client := cluster.Cores[0].Client

	err := client.Sys().Mount("test-backend", &api.MountInput{
		Type: "test-backend",
	})

	if err != nil {
		t.Fatalf("failed to mount test backend: %v", err)
	}

	// Ensure that the public and private routes are accessible through the encrypted listener
	_, err = client.Logical().Read("test-backend/unauthenticated/private")
	require.NoError(t, err, "Could not access path from main listener")

	_, err = client.Logical().Read("test-backend/unauthenticated/public")
	require.NoError(t, err, "Could not access path from main listener")

	publicRouteClient := cluster.Cores[0].PublicRouteClient

	// Ensure that a request via the public_route listener to the private route (not marked as public) is
	// rejected
	_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/private")
	require.Error(t, err, "Access to the private route via the public route listener must be rejected")

	// Ensure that a request to a public path via a public route listener is accepted
	_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/public")
	require.NoError(t, err, "Could not access public path via public route listener")
}
