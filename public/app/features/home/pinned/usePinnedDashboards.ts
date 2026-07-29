import { useCallback, useMemo, useState } from 'react';
import { useAsyncRetry } from 'react-use';

import { getGrafanaSearcher } from 'app/features/search/service/searcher';
import { type DashboardQueryResult } from 'app/features/search/service/types';

import { pinDashboard, listPinnedDashboards, reorderPinnedDashboards, unpinDashboard, updatePinnedDashboardNote } from './api';
import { type PinnedDashboardView } from './types';

function toPinnedViews(pins: Awaited<ReturnType<typeof listPinnedDashboards>>, dashboards: DashboardQueryResult[]) {
  const dashboardByUid = new Map(dashboards.map((dashboard) => [dashboard.uid, dashboard]));

  return pins
    .map((pin) => {
      const dashboard = dashboardByUid.get(pin.dashboardUid);
      if (!dashboard) {
        return null;
      }

      return {
        uid: dashboard.uid,
        name: dashboard.name,
        url: dashboard.url,
        location: dashboard.location,
        note: pin.note,
        sortOrder: pin.sortOrder,
      } satisfies PinnedDashboardView;
    })
    .filter((view): view is PinnedDashboardView => view !== null);
}

export function usePinnedDashboards() {
  const [mutationError, setMutationError] = useState<Error>();

  const {
    value: pinnedDashboards = [],
    loading,
    error,
    retry,
  } = useAsyncRetry(async () => {
    const pins = await listPinnedDashboards();
    if (!pins.length) {
      return [];
    }

    const uids = pins.map((pin) => pin.dashboardUid);
    const searchResults = await getGrafanaSearcher().search({
      kind: ['dashboard'],
      limit: uids.length,
      uid: uids,
    });

    return toPinnedViews(pins, searchResults.view.toArray());
  }, []);

  const pinnedUidSet = useMemo(() => new Set(pinnedDashboards.map((dashboard) => dashboard.uid)), [pinnedDashboards]);

  const isPinned = useCallback((uid: string) => pinnedUidSet.has(uid), [pinnedUidSet]);

  const pin = useCallback(
    async (uid: string, note?: string) => {
      setMutationError(undefined);
      try {
        await pinDashboard(uid, note ? { note } : undefined);
        retry();
      } catch (err) {
        setMutationError(err instanceof Error ? err : new Error('Failed to pin dashboard'));
        throw err;
      }
    },
    [retry]
  );

  const unpin = useCallback(
    async (uid: string) => {
      setMutationError(undefined);
      try {
        await unpinDashboard(uid);
        retry();
      } catch (err) {
        setMutationError(err instanceof Error ? err : new Error('Failed to unpin dashboard'));
        throw err;
      }
    },
    [retry]
  );

  const updateNote = useCallback(
    async (uid: string, note: string) => {
      setMutationError(undefined);
      try {
        await updatePinnedDashboardNote(uid, { note });
        retry();
      } catch (err) {
        setMutationError(err instanceof Error ? err : new Error('Failed to update pin note'));
        throw err;
      }
    },
    [retry]
  );

  const reorder = useCallback(
    async (dashboardUids: string[]) => {
      setMutationError(undefined);
      try {
        await reorderPinnedDashboards({ dashboardUids });
        retry();
      } catch (err) {
        setMutationError(err instanceof Error ? err : new Error('Failed to reorder pinned dashboards'));
        throw err;
      }
    },
    [retry]
  );

  return {
    pinnedDashboards,
    loading,
    error: error ?? mutationError,
    retry,
    isPinned,
    pin,
    unpin,
    updateNote,
    reorder,
  };
}
