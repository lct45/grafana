import { getBackendSrv } from '@grafana/runtime';

import {
  createDashboardPin,
  deleteDashboardPin,
  listDashboardPins,
  reorderDashboardPins,
  updateDashboardPin,
} from './api';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn(),
}));

describe('dashboard pins api', () => {
  const backendSrv = {
    post: jest.fn(),
    get: jest.fn(),
    patch: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    jest.mocked(getBackendSrv).mockReturnValue(backendSrv as unknown as ReturnType<typeof getBackendSrv>);
  });

  it('creates a pin', async () => {
    backendSrv.post.mockResolvedValue({
      pin: {
        uid: 'pin123',
        dashboardUid: 'dash123',
        sortOrder: 0,
        note: 'On-call',
        createdAt: 1,
        updatedAt: 1,
      },
    });

    const result = await createDashboardPin({ dashboardUid: 'dash123', note: 'On-call' });

    expect(backendSrv.post).toHaveBeenCalledWith('/api/home/dashboard-pins', {
      dashboardUid: 'dash123',
      note: 'On-call',
    });
    expect(result.uid).toBe('pin123');
  });

  it('lists pins', async () => {
    backendSrv.get.mockResolvedValue({
      pins: [{ uid: 'pin123', dashboardUid: 'dash123', sortOrder: 0, createdAt: 1, updatedAt: 1 }],
    });

    const result = await listDashboardPins();

    expect(backendSrv.get).toHaveBeenCalledWith('/api/home/dashboard-pins');
    expect(result).toHaveLength(1);
  });

  it('updates a pin note', async () => {
    backendSrv.patch.mockResolvedValue({
      pin: { uid: 'pin123', dashboardUid: 'dash123', sortOrder: 0, note: 'Updated', createdAt: 1, updatedAt: 2 },
    });

    const result = await updateDashboardPin('pin123', { note: 'Updated' });

    expect(backendSrv.patch).toHaveBeenCalledWith('/api/home/dashboard-pins/pin123', { note: 'Updated' });
    expect(result.note).toBe('Updated');
  });

  it('reorders pins', async () => {
    await reorderDashboardPins({ uids: ['pin2', 'pin1'] });
    expect(backendSrv.put).toHaveBeenCalledWith('/api/home/dashboard-pins/reorder', { uids: ['pin2', 'pin1'] });
  });

  it('deletes a pin', async () => {
    await deleteDashboardPin('pin123');
    expect(backendSrv.delete).toHaveBeenCalledWith('/api/home/dashboard-pins/pin123');
  });
});
