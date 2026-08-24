package dashboardpins

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/grafana/grafana/pkg/services/user"
	"github.com/grafana/grafana/pkg/tests/testsuite"
)

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

const (
	testOrgID   = int64(1)
	testOrgID2  = int64(2)
	testUserID  = int64(1)
	testDashUID = "test-dashboard-uid"
)

type mockDashboardService struct {
	existing map[int64]map[string]bool
}

func (m *mockDashboardService) GetDashboard(_ context.Context, query *dashboards.GetDashboardQuery) (*dashboards.Dashboard, error) {
	orgDashboards, ok := m.existing[query.OrgID]
	if !ok {
		return nil, dashboards.ErrDashboardNotFound
	}
	if orgDashboards[query.UID] {
		return &dashboards.Dashboard{UID: query.UID, OrgID: query.OrgID}, nil
	}
	return nil, dashboards.ErrDashboardNotFound
}

func newTestService(t *testing.T, orgDashboards map[int64][]string) (*DashboardPinsService, db.DB) {
	t.Helper()

	sqlStore, _ := db.InitTestDBWithCfg(t)
	existing := make(map[int64]map[string]bool)
	for orgID, uids := range orgDashboards {
		existing[orgID] = make(map[string]bool, len(uids))
		for _, uid := range uids {
			existing[orgID][uid] = true
		}
	}

	service := NewTestService(sqlStore, &mockDashboardService{existing: existing}, time.Now)
	return service, sqlStore
}

func testUser(userID, orgID int64) *user.SignedInUser {
	return &user.SignedInUser{
		UserID:  userID,
		OrgID:   orgID,
		OrgRole: org.RoleEditor,
	}
}

func notePtr(value string) *string {
	return &value
}

func TestCreatePin(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	pin, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
		Note:         notePtr("Primary dashboard"),
	})
	require.NoError(t, err)
	require.Equal(t, testDashUID, pin.DashboardUID)
	require.NotNil(t, pin.Note)
	require.Equal(t, "Primary dashboard", *pin.Note)
	require.Equal(t, 0, pin.SortOrder)
	require.NotZero(t, pin.CreatedAt)
}

func TestCreatePinAppendsSortOrder(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID, "second-dashboard"}})

	first, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)
	require.Equal(t, 0, first.SortOrder)

	second, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: "second-dashboard",
	})
	require.NoError(t, err)
	require.Equal(t, 1, second.SortOrder)
}

func TestCreatePinValidation(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{})
	require.ErrorIs(t, err, ErrDashboardUIDRequired)

	longNote := string(make([]byte, MaxNoteLength+1))
	_, err = service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
		Note:         &longNote,
	})
	require.ErrorIs(t, err, ErrNoteTooLong)
}

func TestCreatePinDashboardNotFound(t *testing.T) {
	service, _ := newTestService(t, nil)

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.ErrorIs(t, err, ErrDashboardNotFound)
}

func TestCreatePinAlreadyExists(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)

	_, err = service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.ErrorIs(t, err, ErrPinAlreadyExists)
}

func TestCreatePinLimitReached(t *testing.T) {
	dashboardUIDs := make([]string, MaxPins+1)
	for i := range dashboardUIDs {
		dashboardUIDs[i] = fmt.Sprintf("dash%d", i)
	}

	service, _ := newTestService(t, map[int64][]string{testOrgID: dashboardUIDs})

	for i := 0; i < MaxPins; i++ {
		_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
			DashboardUID: dashboardUIDs[i],
		})
		require.NoError(t, err)
	}

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: dashboardUIDs[MaxPins],
	})
	require.ErrorIs(t, err, ErrPinLimitReached)
}

func TestListPinsOrderedBySortOrder(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID, "second-dashboard"}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)
	_, err = service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: "second-dashboard",
	})
	require.NoError(t, err)

	_, err = service.ReorderPins(context.Background(), testUser(testUserID, testOrgID), ReorderDashboardPinsCommand{
		DashboardUIDs: []string{"second-dashboard", testDashUID},
	})
	require.NoError(t, err)

	pins, err := service.ListPins(context.Background(), testUser(testUserID, testOrgID))
	require.NoError(t, err)
	require.Len(t, pins, 2)
	require.Equal(t, "second-dashboard", pins[0].DashboardUID)
	require.Equal(t, 0, pins[0].SortOrder)
	require.Equal(t, testDashUID, pins[1].DashboardUID)
	require.Equal(t, 1, pins[1].SortOrder)
}

func TestReorderPinsTrimsUIDs(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID, "second-dashboard"}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)
	_, err = service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: "second-dashboard",
	})
	require.NoError(t, err)

	pins, err := service.ReorderPins(context.Background(), testUser(testUserID, testOrgID), ReorderDashboardPinsCommand{
		DashboardUIDs: []string{"  second-dashboard  ", "  " + testDashUID + "  "},
	})
	require.NoError(t, err)
	require.Equal(t, "second-dashboard", pins[0].DashboardUID)
	require.Equal(t, testDashUID, pins[1].DashboardUID)
}

func TestReorderPinsRequiresExactUIDSet(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID, "second-dashboard"}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)
	_, err = service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: "second-dashboard",
	})
	require.NoError(t, err)

	_, err = service.ReorderPins(context.Background(), testUser(testUserID, testOrgID), ReorderDashboardPinsCommand{
		DashboardUIDs: []string{testDashUID},
	})
	require.ErrorIs(t, err, ErrInvalidReorder)

	_, err = service.ReorderPins(context.Background(), testUser(testUserID, testOrgID), ReorderDashboardPinsCommand{
		DashboardUIDs: []string{testDashUID, "second-dashboard", "missing-dashboard"},
	})
	require.ErrorIs(t, err, ErrInvalidReorder)
}

func TestPatchPinNote(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)

	updated, err := service.PatchPin(context.Background(), testUser(testUserID, testOrgID), testDashUID, PatchDashboardPinCommand{
		Note: notePtr("Updated note"),
	})
	require.NoError(t, err)
	require.NotNil(t, updated.Note)
	require.Equal(t, "Updated note", *updated.Note)
}

func TestPatchPinClearsNote(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
		Note:         notePtr("Initial note"),
	})
	require.NoError(t, err)

	updated, err := service.PatchPin(context.Background(), testUser(testUserID, testOrgID), testDashUID, PatchDashboardPinCommand{
		Note: nil,
	})
	require.NoError(t, err)
	require.Nil(t, updated.Note)
}

func TestPatchPinNotFound(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	_, err := service.PatchPin(context.Background(), testUser(testUserID, testOrgID), testDashUID, PatchDashboardPinCommand{
		Note: notePtr("Missing pin"),
	})
	require.ErrorIs(t, err, ErrPinNotFound)
}

func TestDeletePin(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)

	err = service.DeletePin(context.Background(), testUser(testUserID, testOrgID), testDashUID)
	require.NoError(t, err)

	pins, err := service.ListPins(context.Background(), testUser(testUserID, testOrgID))
	require.NoError(t, err)
	require.Empty(t, pins)
}

func TestDeletePinNotFound(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	err := service.DeletePin(context.Background(), testUser(testUserID, testOrgID), testDashUID)
	require.ErrorIs(t, err, ErrPinNotFound)
}

func TestPinsAreUserScoped(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{testOrgID: {testDashUID}})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)

	pins, err := service.ListPins(context.Background(), testUser(999, testOrgID))
	require.NoError(t, err)
	require.Empty(t, pins)

	err = service.DeletePin(context.Background(), testUser(999, testOrgID), testDashUID)
	require.ErrorIs(t, err, ErrPinNotFound)
}

func TestPinsAreOrgScoped(t *testing.T) {
	service, _ := newTestService(t, map[int64][]string{
		testOrgID:  {testDashUID},
		testOrgID2: {testDashUID},
	})

	_, err := service.CreatePin(context.Background(), testUser(testUserID, testOrgID), CreateDashboardPinCommand{
		DashboardUID: testDashUID,
	})
	require.NoError(t, err)

	pins, err := service.ListPins(context.Background(), testUser(testUserID, testOrgID2))
	require.NoError(t, err)
	require.Empty(t, pins)

	err = service.DeletePin(context.Background(), testUser(testUserID, testOrgID2), testDashUID)
	require.ErrorIs(t, err, ErrPinNotFound)
}

func TestValidateDashboardExistsPropagatesUnexpectedError(t *testing.T) {
	service := &DashboardPinsService{
		dashboardService: &unexpectedDashboardService{},
	}

	err := service.validateDashboardExists(context.Background(), testOrgID, testDashUID)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrDashboardNotFound))
}

type unexpectedDashboardService struct{}

func (unexpectedDashboardService) GetDashboard(_ context.Context, _ *dashboards.GetDashboardQuery) (*dashboards.Dashboard, error) {
	return nil, errors.New("unexpected failure")
}
