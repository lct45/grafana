package dashboardpinimpl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/dashboardpin"
	"github.com/grafana/grafana/pkg/tests/testsuite"
	"github.com/grafana/grafana/pkg/util/testutil"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

type getStore func(db.DB) store

func testIntegrationDashboardPinDataAccess(t *testing.T, fn getStore) {
	t.Helper()

	t.Run("Testing Dashboard Pin Data Access", func(t *testing.T) {
		ss := db.InitTestDB(t)
		pinStore := fn(ss)

		t.Run("Given saved pin by dashboard UID", func(t *testing.T) {
			cmd := dashboardpin.PinDashboardCommand{
				DashboardUID: "test-dashboard",
				OrgID:        1,
				UserID:       12,
				Note:         "On-call overview",
			}
			pin, err := pinStore.Insert(context.Background(), &cmd, 0)
			require.NoError(t, err)
			require.Equal(t, 0, pin.SortOrder)
			require.Equal(t, "On-call overview", pin.Note)

			t.Run("Get should return the pin", func(t *testing.T) {
				got, err := pinStore.Get(context.Background(), 12, 1, "test-dashboard")
				require.NoError(t, err)
				require.Equal(t, "test-dashboard", got.DashboardUID)
			})

			t.Run("List should return a list of size 1", func(t *testing.T) {
				result, err := pinStore.List(context.Background(), &dashboardpin.ListPinsQuery{UserID: 12, OrgID: 1})
				require.NoError(t, err)
				require.Equal(t, 1, len(result))
			})

			t.Run("Update note should persist", func(t *testing.T) {
				updated, err := pinStore.UpdateNote(context.Background(), &dashboardpin.UpdatePinNoteCommand{
					UserID:       12,
					OrgID:        1,
					DashboardUID: "test-dashboard",
					Note:         "Updated note",
				})
				require.NoError(t, err)
				require.Equal(t, "Updated note", updated.Note)
			})

			t.Run("Delete should remove the pin", func(t *testing.T) {
				err := pinStore.Delete(context.Background(), &dashboardpin.UnpinDashboardCommand{
					DashboardUID: "test-dashboard",
					OrgID:        1,
					UserID:       12,
				})
				require.NoError(t, err)

				_, err = pinStore.Get(context.Background(), 12, 1, "test-dashboard")
				require.ErrorIs(t, err, dashboardpin.ErrPinNotFound)
			})
		})

		t.Run("Reorder should update sort order", func(t *testing.T) {
			_, err := pinStore.Insert(context.Background(), &dashboardpin.PinDashboardCommand{
				DashboardUID: "first",
				OrgID:        1,
				UserID:       20,
			}, 0)
			require.NoError(t, err)
			_, err = pinStore.Insert(context.Background(), &dashboardpin.PinDashboardCommand{
				DashboardUID: "second",
				OrgID:        1,
				UserID:       20,
			}, 1)
			require.NoError(t, err)

			err = pinStore.Reorder(context.Background(), &dashboardpin.ReorderPinsCommand{
				UserID:        20,
				OrgID:         1,
				DashboardUIDs: []string{"second", "first"},
			})
			require.NoError(t, err)

			pins, err := pinStore.List(context.Background(), &dashboardpin.ListPinsQuery{UserID: 20, OrgID: 1})
			require.NoError(t, err)
			require.Equal(t, "second", pins[0].DashboardUID)
			require.Equal(t, 0, pins[0].SortOrder)
			require.Equal(t, "first", pins[1].DashboardUID)
			require.Equal(t, 1, pins[1].SortOrder)
		})

		t.Run("DeleteByUser should remove pins for user", func(t *testing.T) {
			_, err := pinStore.Insert(context.Background(), &dashboardpin.PinDashboardCommand{
				DashboardUID: "user-pin",
				OrgID:        1,
				UserID:       30,
			}, 0)
			require.NoError(t, err)
			_, err = pinStore.Insert(context.Background(), &dashboardpin.PinDashboardCommand{
				DashboardUID: "other-user-pin",
				OrgID:        1,
				UserID:       31,
			}, 0)
			require.NoError(t, err)

			err = pinStore.DeleteByUser(context.Background(), 30)
			require.NoError(t, err)

			pins, err := pinStore.List(context.Background(), &dashboardpin.ListPinsQuery{UserID: 30, OrgID: 1})
			require.NoError(t, err)
			require.Empty(t, pins)

			pins, err = pinStore.List(context.Background(), &dashboardpin.ListPinsQuery{UserID: 31, OrgID: 1})
			require.NoError(t, err)
			require.Len(t, pins, 1)
		})
	})
}

func TestIntegrationSQLStoreDashboardPin(t *testing.T) {
	testutil.SkipIntegrationTestInShortMode(t)
	testIntegrationDashboardPinDataAccess(t, func(db db.DB) store {
		return &sqlStore{db: db}
	})
}
