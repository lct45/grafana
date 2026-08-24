import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';

import { t } from '@grafana/i18n';
import { useAppNotification } from 'app/core/copy/appNotification';
import { contextSrv } from 'app/core/services/context_srv';

import {
  createDashboardPin,
  deleteDashboardPin,
  listDashboardPins,
  patchDashboardPin,
  reorderDashboardPins,
} from './api';
import { type DashboardPin, MAX_PIN_NOTE_LENGTH } from './types';

interface DashboardPinsContextValue {
  pins: DashboardPin[];
  isLoading: boolean;
  error: string | null;
  isPinned: (dashboardUid: string) => boolean;
  refreshPins: () => Promise<void>;
  pinDashboard: (dashboardUid: string, note?: string) => Promise<void>;
  unpinDashboard: (dashboardUid: string) => Promise<void>;
  reorderPins: (dashboardUids: string[]) => Promise<void>;
  updatePinNote: (dashboardUid: string, note: string | null) => Promise<void>;
}

const defaultDashboardPinsContext: DashboardPinsContextValue = {
  pins: [],
  isLoading: false,
  error: null,
  isPinned: () => false,
  refreshPins: async () => {},
  pinDashboard: async () => {},
  unpinDashboard: async () => {},
  reorderPins: async () => {},
  updatePinNote: async () => {},
};

const DashboardPinsContext = createContext<DashboardPinsContextValue>(defaultDashboardPinsContext);

function sortPins(pins: DashboardPin[]) {
  return [...pins].sort((a, b) => a.sortOrder - b.sortOrder || a.createdAt - b.createdAt);
}

function useDashboardPinsState(): DashboardPinsContextValue {
  const notifyApp = useAppNotification();
  const [pins, setPins] = useState<DashboardPin[]>([]);
  const [isLoading, setIsLoading] = useState(contextSrv.isSignedIn);
  const [error, setError] = useState<string | null>(null);

  const refreshPins = useCallback(async () => {
    if (!contextSrv.isSignedIn) {
      setPins([]);
      setIsLoading(false);
      setError(null);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const result = await listDashboardPins();
      setPins(sortPins(result));
    } catch (err) {
      setError(err instanceof Error ? err.message : t('home.pinned-dashboards.load-error', 'Failed to load pins'));
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshPins();
  }, [refreshPins]);

  const isPinned = useCallback((dashboardUid: string) => pins.some((pin) => pin.dashboardUid === dashboardUid), [pins]);

  const pinDashboard = useCallback(
    async (dashboardUid: string, note?: string) => {
      const pin = await createDashboardPin({ dashboardUid, note });
      setPins((current) => sortPins([...current.filter((item) => item.dashboardUid !== dashboardUid), pin]));
      notifyApp.success(t('home.pinned-dashboards.pin-success', 'Dashboard pinned to Home'));
    },
    [notifyApp]
  );

  const unpinDashboard = useCallback(
    async (dashboardUid: string) => {
      await deleteDashboardPin(dashboardUid);
      setPins((current) => current.filter((pin) => pin.dashboardUid !== dashboardUid));
      notifyApp.success(t('home.pinned-dashboards.unpin-success', 'Dashboard unpinned from Home'));
    },
    [notifyApp]
  );

  const reorderPins = useCallback(async (dashboardUids: string[]) => {
    const result = await reorderDashboardPins({ dashboardUids });
    setPins(sortPins(result));
  }, []);

  const updatePinNote = useCallback(async (dashboardUid: string, note: string | null) => {
    const trimmed = note?.trim() ?? '';
    const normalized = trimmed.length > 0 ? trimmed.slice(0, MAX_PIN_NOTE_LENGTH) : null;
    const updated = await patchDashboardPin(dashboardUid, { note: normalized });
    setPins((current) => sortPins(current.map((pin) => (pin.dashboardUid === dashboardUid ? updated : pin))));
  }, []);

  return useMemo(
    () => ({
      pins,
      isLoading,
      error,
      isPinned,
      refreshPins,
      pinDashboard,
      unpinDashboard,
      reorderPins,
      updatePinNote,
    }),
    [error, isLoading, isPinned, pinDashboard, pins, refreshPins, reorderPins, unpinDashboard, updatePinNote]
  );
}

export function DashboardPinsProvider({ children }: { children: ReactNode }) {
  const value = useDashboardPinsState();
  return <DashboardPinsContext.Provider value={value}>{children}</DashboardPinsContext.Provider>;
}

export function useDashboardPins() {
  return useContext(DashboardPinsContext);
}
