import { useCallback, useEffect, useMemo, useState } from 'react';

import {
  createDashboardPin,
  deleteDashboardPin,
  listDashboardPins,
  reorderDashboardPins,
  updateDashboardPin,
} from './api';
import { type DashboardPin } from './types';

export function useDashboardPins() {
  const [pins, setPins] = useState<DashboardPin[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error>();

  const refresh = useCallback(async () => {
    setIsLoading(true);
    setError(undefined);
    try {
      const nextPins = await listDashboardPins();
      setPins(nextPins);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to load dashboard pins'));
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const pinnedDashboardUids = useMemo(() => new Set(pins.map((pin) => pin.dashboardUid)), [pins]);

  const pinDashboard = useCallback(
    async (dashboardUid: string, note?: string) => {
      const pin = await createDashboardPin({ dashboardUid, note });
      setPins((current) => [...current, pin]);
      return pin;
    },
    []
  );

  const unpinDashboard = useCallback(async (pinUid: string) => {
    await deleteDashboardPin(pinUid);
    setPins((current) => current.filter((pin) => pin.uid !== pinUid));
  }, []);

  const updatePinNote = useCallback(async (pinUid: string, note: string) => {
    const pin = await updateDashboardPin(pinUid, { note });
    setPins((current) => current.map((existing) => (existing.uid === pinUid ? pin : existing)));
    return pin;
  }, []);

  const reorderPins = useCallback(async (uids: string[]) => {
    await reorderDashboardPins({ uids });
    setPins((current) => {
      const byUid = new Map(current.map((pin) => [pin.uid, pin]));
      return uids.flatMap((uid, index) => {
        const pin = byUid.get(uid);
        return pin ? [{ ...pin, sortOrder: index }] : [];
      });
    });
  }, []);

  const isPinned = useCallback((dashboardUid: string) => pinnedDashboardUids.has(dashboardUid), [pinnedDashboardUids]);

  const getPinUid = useCallback(
    (dashboardUid: string) => pins.find((pin) => pin.dashboardUid === dashboardUid)?.uid,
    [pins]
  );

  return {
    pins,
    isLoading,
    error,
    refresh,
    pinDashboard,
    unpinDashboard,
    updatePinNote,
    reorderPins,
    isPinned,
    getPinUid,
  };
}
