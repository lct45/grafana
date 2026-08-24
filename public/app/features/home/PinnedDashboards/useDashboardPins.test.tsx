import { act, renderHook, waitFor } from '@testing-library/react';
import { type ReactNode } from 'react';

import { contextSrv } from 'app/core/services/context_srv';

import * as dashboardPinsApi from './api';
import { DashboardPinsProvider, useDashboardPins } from './useDashboardPins';

jest.mock('./api');

jest.mock('app/core/copy/appNotification', () => ({
  useAppNotification: () => ({
    success: jest.fn(),
    error: jest.fn(),
  }),
}));

function wrapper({ children }: { children: ReactNode }) {
  return <DashboardPinsProvider>{children}</DashboardPinsProvider>;
}

describe('useDashboardPins', () => {
  let originalSignedIn: boolean;

  beforeEach(() => {
    originalSignedIn = contextSrv.isSignedIn;
    contextSrv.isSignedIn = true;
    jest.clearAllMocks();
    jest.spyOn(dashboardPinsApi, 'listDashboardPins').mockResolvedValue([
      { dashboardUid: 'dash-1', sortOrder: 0, createdAt: 1 },
      { dashboardUid: 'dash-2', note: 'Ops', sortOrder: 1, createdAt: 2 },
    ]);
    jest.spyOn(dashboardPinsApi, 'createDashboardPin').mockResolvedValue({
      dashboardUid: 'dash-3',
      sortOrder: 2,
      createdAt: 3,
    });
    jest.spyOn(dashboardPinsApi, 'deleteDashboardPin').mockResolvedValue();
    jest.spyOn(dashboardPinsApi, 'reorderDashboardPins').mockResolvedValue([
      { dashboardUid: 'dash-2', note: 'Ops', sortOrder: 0, createdAt: 2 },
      { dashboardUid: 'dash-1', sortOrder: 1, createdAt: 1 },
    ]);
    jest.spyOn(dashboardPinsApi, 'patchDashboardPin').mockResolvedValue({
      dashboardUid: 'dash-1',
      note: 'Updated note',
      sortOrder: 0,
      createdAt: 1,
    });
  });

  afterEach(() => {
    contextSrv.isSignedIn = originalSignedIn;
  });

  it('loads pins on mount', async () => {
    const { result } = renderHook(() => useDashboardPins(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.pins).toHaveLength(2);
    expect(result.current.isPinned('dash-2')).toBe(true);
  });

  it('pins and unpins dashboards', async () => {
    const { result } = renderHook(() => useDashboardPins(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await act(async () => {
      await result.current.pinDashboard('dash-3');
    });

    expect(dashboardPinsApi.createDashboardPin).toHaveBeenCalledWith({ dashboardUid: 'dash-3' });
    expect(result.current.isPinned('dash-3')).toBe(true);

    await act(async () => {
      await result.current.unpinDashboard('dash-3');
    });

    expect(dashboardPinsApi.deleteDashboardPin).toHaveBeenCalledWith('dash-3');
    expect(result.current.isPinned('dash-3')).toBe(false);
  });

  it('reorders and updates notes', async () => {
    const { result } = renderHook(() => useDashboardPins(), { wrapper });

    await waitFor(() => {
      expect(result.current.pins).toHaveLength(2);
    });

    await act(async () => {
      await result.current.reorderPins(['dash-2', 'dash-1']);
    });

    expect(dashboardPinsApi.reorderDashboardPins).toHaveBeenCalledWith({ dashboardUids: ['dash-2', 'dash-1'] });
    expect(result.current.pins[0].dashboardUid).toBe('dash-2');

    await act(async () => {
      await result.current.updatePinNote('dash-1', 'Updated note');
    });

    expect(dashboardPinsApi.patchDashboardPin).toHaveBeenCalledWith('dash-1', { note: 'Updated note' });
    expect(result.current.pins.find((pin) => pin.dashboardUid === 'dash-1')?.note).toBe('Updated note');
  });
});
