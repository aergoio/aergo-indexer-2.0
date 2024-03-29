package db

import (
	"context"
	"testing"

	doc "github.com/aergoio/aergo-indexer-2.0/indexer/documents"
	"github.com/orlangure/gnomock"
	mockElastic "github.com/orlangure/gnomock/preset/elastic"
	"github.com/stretchr/testify/require"
)

func TestElastic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode.")
	}

	mock, err := mockupDocker()
	require.NoError(t, err)
	defer gnomock.Stop(mock)
	doc.InitEsMappings(false)

	TestDatabaseSuite(t, func() DbController {
		ctx := context.Background()
		dbController, err := NewElasticsearchDbController(ctx, mock.DefaultAddress())
		require.NoError(t, err)
		_, err = dbController.client.DeleteIndex("*").Do(ctx)
		require.NoError(t, err)
		return dbController
	})
}

func mockupDocker() (mock *gnomock.Container, err error) {
	preset := mockElastic.Preset(
		// version 7 official preset ( github.com/orlangure/gnomock#official-presets )
		mockElastic.WithVersion("7.9.3"),
	)
	mock, err = gnomock.Start(
		preset,
		gnomock.WithEnv("discovery.type=single-node"),
		gnomock.WithEnv("bootstrap.memory_lock=true"),
		gnomock.WithEnv("ES_JAVA_OPTS=-Xms512m -Xmx512m"),
		gnomock.WithEnv("xpack.security.enabled=false"),
	)
	if err != nil {
		return nil, err
	}
	return mock, nil
}
