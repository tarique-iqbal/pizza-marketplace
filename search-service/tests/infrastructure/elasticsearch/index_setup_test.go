package elasticsearch_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	esinfra "search-service/internal/infrastructure/elasticsearch"
	"search-service/tests/testutil"
)

func TestEnsureIndex_IdempotentOnSecondCall(t *testing.T) {
	es := testutil.ES(t)

	require.NoError(t, esinfra.EnsureIndex(context.Background(), es))
	require.NoError(t, esinfra.EnsureIndex(context.Background(), es))
}
