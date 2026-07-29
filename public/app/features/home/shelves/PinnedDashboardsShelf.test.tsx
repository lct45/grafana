import { http, HttpResponse } from 'msw';
import { render, screen } from 'test/test-utils';

import { type DashboardHit } from '@grafana/api-clients/rtkq/dashboard/v0alpha1';
import { setBackendSrv } from '@grafana/runtime';
import { getCustomSearchHandler } from '@grafana/test-utils/handlers';
import server, { setupMockServer } from '@grafana/test-utils/server';
import { backendSrv } from 'app/core/services/backend_srv';

import { PinnedDashboardsShelf } from './PinnedDashboardsShelf';

setBackendSrv(backendSrv);
setupMockServer();

function makeDashboardHit(overrides: Partial<DashboardHit> & { name: string; title: string }): DashboardHit {
  return {
    resource: 'dashboards',
    folder: 'general',
    field: {},
    ...overrides,
  };
}

const pinnedHits: DashboardHit[] = [
  makeDashboardHit({ name: 'pinned-1', title: 'Pinned Dashboard 1' }),
  makeDashboardHit({ name: 'pinned-2', title: 'Pinned Dashboard 2' }),
];

beforeEach(() => {
  server.use(
    getCustomSearchHandler(pinnedHits),
    http.put('/api/user/pinned-dashboards/order', () => HttpResponse.json({ message: 'Pinned dashboards reordered' })),
    http.patch('/api/user/pinned-dashboards/dashboard/uid/:uid', () =>
      HttpResponse.json({
        dashboardUid: 'pinned-1',
        sortOrder: 0,
        note: 'Updated note',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-02T00:00:00Z',
      })
    ),
    http.delete('/api/user/pinned-dashboards/dashboard/uid/:uid', () =>
      HttpResponse.json({ message: 'Dashboard unpinned' })
    )
  );
});

describe('PinnedDashboardsShelf', () => {
  it('renders pinned dashboards', async () => {
    render(
      <PinnedDashboardsShelf
        dashboards={[
          {
            uid: 'pinned-1',
            name: 'Pinned Dashboard 1',
            url: '/d/pinned-1',
            location: 'general',
            sortOrder: 0,
          },
        ]}
        loading={false}
        error={undefined}
        retry={jest.fn()}
        foldersByUid={{}}
        onUnpin={jest.fn()}
        onReorder={jest.fn()}
        onUpdateNote={jest.fn()}
      />
    );

    expect(await screen.findByText('Pinned Dashboard 1')).toBeInTheDocument();
  });

  it('renders empty state when there are no pins', async () => {
    render(
      <PinnedDashboardsShelf
        dashboards={[]}
        loading={false}
        error={undefined}
        retry={jest.fn()}
        foldersByUid={{}}
        onUnpin={jest.fn()}
        onReorder={jest.fn()}
        onUpdateNote={jest.fn()}
      />
    );

    expect(await screen.findByText('Pinned dashboards will appear here.')).toBeInTheDocument();
  });
});
