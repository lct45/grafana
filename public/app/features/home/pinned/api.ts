import { getBackendSrv } from '@grafana/runtime';

import {
  type PinDashboardRequest,
  type PinnedDashboardRecord,
  type ReorderPinnedDashboardsRequest,
  type UpdatePinNoteRequest,
} from './types';

const BASE_URL = '/api/user/pinned-dashboards';

export function listPinnedDashboards(): Promise<PinnedDashboardRecord[]> {
  return getBackendSrv().get<PinnedDashboardRecord[]>(BASE_URL);
}

export function pinDashboard(uid: string, body?: PinDashboardRequest): Promise<PinnedDashboardRecord> {
  return getBackendSrv().post(`${BASE_URL}/dashboard/uid/${uid}`, body ?? {});
}

export function unpinDashboard(uid: string): Promise<{ message: string }> {
  return getBackendSrv().delete(`${BASE_URL}/dashboard/uid/${uid}`);
}

export function updatePinnedDashboardNote(uid: string, body: UpdatePinNoteRequest): Promise<PinnedDashboardRecord> {
  return getBackendSrv().patch(`${BASE_URL}/dashboard/uid/${uid}`, body);
}

export function reorderPinnedDashboards(body: ReorderPinnedDashboardsRequest): Promise<{ message: string }> {
  return getBackendSrv().put(`${BASE_URL}/order`, body);
}
