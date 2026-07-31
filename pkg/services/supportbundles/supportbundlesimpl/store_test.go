package supportbundlesimpl

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/kvstore"
	"github.com/grafana/grafana/pkg/services/supportbundles"
	"github.com/grafana/grafana/pkg/services/user"
)

func TestStore_GetNotFound(t *testing.T) {
	s := newStore(kvstore.NewFakeKVStore())

	bundle, err := s.Get(context.Background(), "missing-uid")
	require.Error(t, err)
	assert.Nil(t, bundle)
	assert.True(t, errors.Is(err, supportbundles.ErrNotFound))
}

func TestStore_GetExisting(t *testing.T) {
	s := newStore(kvstore.NewFakeKVStore())

	created, err := s.Create(context.Background(), &user.SignedInUser{UserID: 1, Login: "alice"})
	require.NoError(t, err)

	bundle, err := s.Get(context.Background(), created.UID)
	require.NoError(t, err)
	assert.Equal(t, created.UID, bundle.UID)
	assert.Equal(t, "alice", bundle.Creator)
}

func TestStore_RemoveNotFound(t *testing.T) {
	s := &Service{
		store: newStore(kvstore.NewFakeKVStore()),
	}

	err := s.remove(context.Background(), "missing-uid")
	require.Error(t, err)
	assert.True(t, errors.Is(err, supportbundles.ErrNotFound))
}
