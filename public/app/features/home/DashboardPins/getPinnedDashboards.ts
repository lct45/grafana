import { getGrafanaSearcher } from 'app/features/search/service/searcher';
import { type DashboardQueryResult } from 'app/features/search/service/types';

import { listDashboardPins } from './api';
import { type DashboardPin, type PinnedDashboardItem } from './types';

export async function getPinnedDashboards(): Promise<PinnedDashboardItem[]> {
  try {
    const pins = await listDashboardPins();
    if (!pins.length) {
      return [];
    }

    const dashboardUids = pins.map((pin) => pin.dashboardUid);
    const searchResults = await getGrafanaSearcher().search({
      kind: ['dashboard'],
      limit: dashboardUids.length,
      uid: dashboardUids,
    });

    const dashboardsByUid = new Map(searchResults.view.toArray().map((dashboard) => [dashboard.uid, dashboard]));
    const order = (pin: DashboardPin) => pin.sortOrder;

    return pins
      .slice()
      .sort((a, b) => order(a) - order(b))
      .flatMap((pin) => {
        const dashboard = dashboardsByUid.get(pin.dashboardUid);
        if (!dashboard) {
          return [];
        }

        return [toPinnedDashboardItem(pin, dashboard)];
      });
  } catch (error) {
    console.error('Failed to load pinned dashboards', error);
    return [];
  }
}

function toPinnedDashboardItem(pin: DashboardPin, dashboard: DashboardQueryResult): PinnedDashboardItem {
  return {
    pinUid: pin.uid,
    note: pin.note,
    dashboardUid: pin.dashboardUid,
    name: dashboard.name,
    uid: dashboard.uid,
    url: dashboard.url,
    location: dashboard.location,
    kind: dashboard.kind,
  };
}
