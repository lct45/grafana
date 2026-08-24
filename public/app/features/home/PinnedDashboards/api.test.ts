import { getBackendSrv } from '@grafana/runtime';

import {
  createDashboardPin,
  deleteDashboardPin,
  listDashboardPins,
  patchDashboardPin,
  reorderDashboardPins,
} from './api';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn(),
}));

describe('dashboard pins api', () => {
  const backendSrv = {
    post: jest.fn(),
    get: jest.fn(),
    put: jest.fn(),
    patch: jest.fn(),
    delete: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    jest.mocked(getBackendSrv).mockReturnValue(backendSrv as unknown as ReturnType<typeof getBackendSrv>);
  });

  it('lists dashboard pins', async () => {
    backendSrv.get.mockResolvedValue({
      pins: [{ dashboardUid: 'dash-1', sortOrder: 0, createdAt: 1 }],
    });

    const result = await listDashboardPins();

    expect(backendSrv.get).toHaveBeenCalledWith('/api/user/dashboard-pins');
    expect(result).toHaveLength(1);
    expect(result[0].dashboardUid).toBe('dash-1');
  });

  it('creates a dashboard pin', async () => {
    backendSrv.post.mockResolvedValue({
      pin: { dashboardUid: 'dash-1', note: 'Weekly review', sortOrder: 0, createdAt: 1 },
    });

    const result = await createDashboardPin({ dashboardUid: 'dash-1', note: 'Weekly review' });

    expect(backendSrv.post).toHaveBeenCalledWith('/api/user/dashboard-pins', {
      dashboardUid: 'dash-1',
      note: 'Weekly review',
    });
    expect(result.dashboardUid).toBe('dash-1');
  });

  it('reorders dashboard pins', async () => {
    backendSrv.put.mockResolvedValue({
      pins: [
        { dashboardUid: 'dash-2', sortOrder: 0, createdAt: 2 },
        { dashboardUid: 'dash-1', sortOrder: 1, createdAt: 1 },
      ],
    });

    const result = await reorderDashboardPins({ dashboardUids: ['dash-2', 'dash-1'] });

    expect(backendSrv.put).toHaveBeenCalledWith('/api/user/dashboard-pins', {
      dashboardUids: ['dash-2', 'dash-1'],
    });
    expect(result.map((pin) => pin.dashboardUid)).toEqual(['dash-2', 'dash-1']);
  });

  it('patches a dashboard pin note', async () => {
    backendSrv.patch.mockResolvedValue({
      pin: { dashboardUid: 'dash-1', note: 'Updated', sortOrder: 0, createdAt: 1 },
    });

    const result = await patchDashboardPin('dash-1', { note: 'Updated' });

    expect(backendSrv.patch).toHaveBeenCalledWith('/api/user/dashboard-pins/dash-1', { note: 'Updated' });
    expect(result.note).toBe('Updated');
  });

  it('deletes a dashboard pin', async () => {
    await deleteDashboardPin('dash-1');
    expect(backendSrv.delete).toHaveBeenCalledWith('/api/user/dashboard-pins/dash-1');
  });
});
