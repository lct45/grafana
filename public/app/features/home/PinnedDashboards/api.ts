import { getBackendSrv } from '@grafana/runtime';

import {
  type CreateDashboardPinPayload,
  type CreateDashboardPinResponse,
  type DashboardPin,
  type ListDashboardPinsResponse,
  type PatchDashboardPinPayload,
  type PatchDashboardPinResponse,
  type ReorderDashboardPinsPayload,
} from './types';

const BASE_URL = '/api/user/dashboard-pins';

export async function listDashboardPins(): Promise<DashboardPin[]> {
  const response = await getBackendSrv().get<ListDashboardPinsResponse>(BASE_URL);
  return response.pins ?? [];
}

export async function createDashboardPin(payload: CreateDashboardPinPayload): Promise<DashboardPin> {
  const response = await getBackendSrv().post<CreateDashboardPinResponse>(BASE_URL, payload);
  return response.pin;
}

export async function reorderDashboardPins(payload: ReorderDashboardPinsPayload): Promise<DashboardPin[]> {
  const response = await getBackendSrv().put<ListDashboardPinsResponse>(BASE_URL, payload);
  return response.pins ?? [];
}

export async function patchDashboardPin(
  dashboardUid: string,
  payload: PatchDashboardPinPayload
): Promise<DashboardPin> {
  const response = await getBackendSrv().patch<PatchDashboardPinResponse>(`${BASE_URL}/${dashboardUid}`, payload);
  return response.pin;
}

export async function deleteDashboardPin(dashboardUid: string): Promise<void> {
  await getBackendSrv().delete(`${BASE_URL}/${dashboardUid}`);
}
