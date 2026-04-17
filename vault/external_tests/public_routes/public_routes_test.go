package public_routes

import (
	"fmt"
	"testing"

	"github.com/openbao/openbao/api/v2"
	vaulthttp "github.com/openbao/openbao/http"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/openbao/openbao/vault"
	"github.com/stretchr/testify/require"
)

func TestPublicRoutes_PathAccess(t *testing.T) {
	t.Parallel()
	coreConfig := &vault.CoreConfig{
		LogicalBackends: map[string]logical.Factory{
			"test-backend": vault.TestPublicRoutesBackendFactory,
		},
	}
	cluster := vault.NewTestCluster(t, coreConfig, &vault.TestClusterOptions{
		HandlerFunc:         vaulthttp.Handler,
		PublicRouteListener: true,
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

	// Ensure that the public and private routes are accessible via the tls-protected listener
	{
		_, err = client.Logical().Read("test-backend/unauthenticated/private")
		require.NoError(t, err, "Could not access path from main listener")

		_, err = client.Logical().Read("test-backend/unauthenticated/public")
		require.NoError(t, err, "Could not access path from main listener")
	}

	publicRouteClient := cluster.Cores[0].PublicRouteClient

	// Ensure the that public and private routes cannot be accessed via the public listener since
	// there are no 'allowed_public_paths' that are configured?
	{
		_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/private")
		require.Error(t, err, "Access to the private route via the public route listener must be rejected")

		_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/public")
		require.Error(t, err, "Access to the public route via the public route listener must be rejected")
	}

	// Configure an allowed public path
	_, err = client.Logical().Write("/sys/mounts/test-backend/tune", map[string]interface{}{
		"allowed_public_paths": []string{"unauthenticated/public"},
	})
	require.NoError(t, err, "Could not tune backend 'allowed_public_paths'")

	{
		// Ensure that the private path is still unaccessible via the public listener
		_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/private")
		require.Error(t, err, "Access to the private route via the public route listener must be rejected")

		// Ensure that the public route listener is now accessible via the public listener
		_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/public")
		require.NoError(t, err, "Could not access public path via public route listener")
	}

	// Remove allowed public paths
	_, err = client.Logical().Write("/sys/mounts/test-backend/tune", map[string]interface{}{
		"allowed_public_paths": []string{},
	})
	require.NoError(t, err, "Could not tune backend 'allowed_public_paths'")

	// Ensure that both paths are no longer accessible via the public listener
	{
		_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/private")
		require.Error(t, err, "Access to the private route via the public route listener must be rejected")

		_, err = publicRouteClient.Logical().Read("test-backend/unauthenticated/public")
		require.Error(t, err, "Access to the public route via the public route listener must be rejected")
	}
}

func TestPublicRoutes_Configure(t *testing.T) {
	t.Parallel()
	coreConfig := &vault.CoreConfig{
		LogicalBackends: map[string]logical.Factory{
			"test-backend": vault.TestPublicRoutesBackendFactory,
		},
	}
	cluster := vault.NewTestCluster(t, coreConfig, &vault.TestClusterOptions{
		HandlerFunc: vaulthttp.Handler,
		NumCores:    1,
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

	// Configure list
	validPaths := []string{
		"random",
		"this/that",

		// Wildcard cases
		"+/wildcard/glob2*",
		"end1/+",
		"end2/+/",
		"end3/+/*",
		"middle1/+/bar",
		"middle2/+/+/bar",
		"+/begin",
		"+/around/+/",
	}

	// Configure each path individually
	for _, path := range validPaths {
		_, err = client.Logical().Write("/sys/mounts/test-backend/tune", map[string]interface{}{
			"allowed_public_paths": []string{path},
		})
		require.NoError(t, err, fmt.Sprintf("Could not configure %s as an allowed public path", path))

		tuneConfig, err := client.Logical().Read("/sys/mounts/test-backend/tune")

		require.NoError(t, err, "Could not read tune config")

		require.NotNil(t, tuneConfig, "Could not read tune config")
		require.NotNil(t, tuneConfig.Data, "Could not read tune config")
		require.NotNil(t, tuneConfig.Data["allowed_public_paths"], "Could not read tune config")

		configuredAllowedPublicPaths := tuneConfig.Data["allowed_public_paths"].([]interface{})
		require.Contains(t, configuredAllowedPublicPaths, path)
	}

	// Unset the allowed public paths
	{
		_, err = client.Logical().Write("/sys/mounts/test-backend/tune", map[string]interface{}{
			"allowed_public_paths": []string{},
		})
		require.NoError(t, err, "Could not unset allowed_public_paths")

		tuneConfig, err := client.Logical().Read("/sys/mounts/test-backend/tune")
		require.NoError(t, err, "Could not read tune config")

		if tuneConfig.Data != nil && tuneConfig.Data["allowed_public_paths"] != nil {
			require.Empty(t, tuneConfig.Data["allowed_public_paths"])
		}
	}

	// Configure a list of paths
	{
		_, err = client.Logical().Write("/sys/mounts/test-backend/tune", map[string]interface{}{
			"allowed_public_paths": validPaths,
		})
		require.NoError(t, err, "Could not configure list of allowed_public_paths")

		tuneConfig, err := client.Logical().Read("/sys/mounts/test-backend/tune")

		require.NoError(t, err, "Could not read tune config")

		require.NotNil(t, tuneConfig, "Could not read tune config")
		require.NotNil(t, tuneConfig.Data, "Could not read tune config")
		require.NotNil(t, tuneConfig.Data["allowed_public_paths"], "Could not read tune config")

		configuredAllowedPublicPaths := tuneConfig.Data["allowed_public_paths"].([]interface{})

		for _, path := range validPaths {
			require.Contains(t, configuredAllowedPublicPaths, path)
		}
	}

	// Should reject invalid paths
	invalidPaths := []string{
		// multiple *
		"a*b*c",
		"secret/*/foo*",
		"**",
		"secret/**",
		"foo*bar*",
		"*/foo/*",

		// +* forbidden
		"+*",
		"secret/+*/foo",
		"secret/foo+*",
		"+*/bar",
		"foo/+*",

		// * not at end
		"*foo",
		"secret/*/bar",
		"secret/foo*bar",
		"secret/*foo",
		"*/foo",
		"foo*bar/baz",

		// + adjacent to non-slash
		"secret+",
		"+secret",
		"secret/foo+bar",
		"secret/foo+",
		"secret/+foo",
		"secret/foo+/",
		"foo+bar",
		"a+b",
		"++",

		// combined edge cases
		"secret/+*/foo",
		"secret/*foo*",
		"+*foo",
		"foo+*bar*",
		"secret++/foo",
	}

	for _, path := range invalidPaths {
		_, err = client.Logical().Write("/sys/mounts/test-backend/tune", map[string]interface{}{
			"allowed_public_paths": []string{path},
		})
		require.Error(t, err, fmt.Sprintf("Invalid path was not rejected: %s", path))

		tuneConfig, err := client.Logical().Read("/sys/mounts/test-backend/tune")

		require.NoError(t, err, "Could not read tune config")

		if tuneConfig.Data != nil && tuneConfig.Data["allowed_public_paths"] != nil {
			require.NotEmpty(t, tuneConfig.Data["allowed_public_paths"], path)
		}
	}
}
