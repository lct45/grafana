import userEvent from '@testing-library/user-event';
import { type ReactNode } from 'react';
import { getWrapper, render, screen, waitFor } from 'test/test-utils';

import { contextSrv } from 'app/core/services/context_srv';
import { getGrafanaSearcher } from 'app/features/search/service/searcher';

import { PinnedDashboardsSection } from './PinnedDashboardsSection';
import * as dashboardPinsApi from './api';
import { DashboardPinsProvider } from './useDashboardPins';

jest.mock('./api');
jest.mock('app/features/search/service/searcher', () => ({
  getGrafanaSearcher: jest.fn(),
}));

const mockGetGrafanaSearcher = jest.mocked(getGrafanaSearcher);
const BaseWrapper = getWrapper({ renderWithRouter: true });

function renderSection() {
  return render(<PinnedDashboardsSection />, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <BaseWrapper>
        <DashboardPinsProvider>{children}</DashboardPinsProvider>
      </BaseWrapper>
    ),
  });
}

describe('PinnedDashboardsSection', () => {
  const searchMock = jest.fn();
  let originalSignedIn: boolean;

  beforeEach(() => {
    originalSignedIn = contextSrv.isSignedIn;
    contextSrv.isSignedIn = true;
    jest.clearAllMocks();
    searchMock.mockResolvedValue({
      view: {
        length: 2,
        get: (index: number) =>
          index === 0
            ? { uid: 'dash-1', name: 'Production Overview', url: '/d/dash-1/production-overview' }
            : { uid: 'dash-2', name: 'On-call Board', url: '/d/dash-2/on-call-board' },
      },
    });
    mockGetGrafanaSearcher.mockReturnValue({ search: searchMock } as unknown as ReturnType<typeof getGrafanaSearcher>);

    jest.spyOn(dashboardPinsApi, 'listDashboardPins').mockResolvedValue([
      { dashboardUid: 'dash-1', sortOrder: 0, createdAt: 1 },
      { dashboardUid: 'dash-2', note: 'Weekly', sortOrder: 1, createdAt: 2 },
    ]);
    jest.spyOn(dashboardPinsApi, 'deleteDashboardPin').mockResolvedValue();
    jest.spyOn(dashboardPinsApi, 'reorderDashboardPins').mockResolvedValue([
      { dashboardUid: 'dash-2', note: 'Weekly', sortOrder: 0, createdAt: 2 },
      { dashboardUid: 'dash-1', sortOrder: 1, createdAt: 1 },
    ]);
    jest.spyOn(dashboardPinsApi, 'patchDashboardPin').mockResolvedValue({
      dashboardUid: 'dash-1',
      note: 'Incident review',
      sortOrder: 0,
      createdAt: 1,
    });
  });

  afterEach(() => {
    contextSrv.isSignedIn = originalSignedIn;
  });

  it('renders an empty state when there are no pins', async () => {
    jest.spyOn(dashboardPinsApi, 'listDashboardPins').mockResolvedValue([]);

    renderSection();

    expect(await screen.findByText('Pinned')).toBeInTheDocument();
    expect(screen.getByText('Pin dashboards from the toolbar to keep them here.')).toBeInTheDocument();
  });

  it('hydrates pinned dashboards in server order', async () => {
    renderSection();

    expect(await screen.findByText('Production Overview')).toBeInTheDocument();
    expect(screen.getByText('On-call Board')).toBeInTheDocument();
    expect(screen.getByText('Weekly')).toBeInTheDocument();

    expect(searchMock).toHaveBeenCalledWith({
      kind: ['dashboard'],
      uid: ['dash-1', 'dash-2'],
      limit: 2,
    });
  });

  it('moves a pin down with keyboard controls', async () => {
    const user = userEvent.setup();
    renderSection();

    await screen.findByText('Production Overview');
    const moveDownButtons = screen.getAllByRole('button', { name: /^move down$/i });
    await user.click(moveDownButtons[0]);

    await waitFor(() => {
      expect(dashboardPinsApi.reorderDashboardPins).toHaveBeenCalledWith({ dashboardUids: ['dash-2', 'dash-1'] });
    });
  });

  it('saves a note on blur', async () => {
    const user = userEvent.setup();
    renderSection();

    await screen.findByText('Production Overview');
    await user.click(screen.getByRole('button', { name: /add note/i }));
    const input = screen.getByRole('textbox', { name: /pin note for production overview/i });
    await user.clear(input);
    await user.type(input, 'Incident review');
    await user.tab();

    await waitFor(() => {
      expect(dashboardPinsApi.patchDashboardPin).toHaveBeenCalledWith('dash-1', { note: 'Incident review' });
    });
  });

  it('unpins a dashboard', async () => {
    const user = userEvent.setup();
    renderSection();

    await screen.findByText('Production Overview');
    await user.click(screen.getByRole('button', { name: /unpin production overview from home/i }));

    await waitFor(() => {
      expect(dashboardPinsApi.deleteDashboardPin).toHaveBeenCalledWith('dash-1');
    });
  });
});
