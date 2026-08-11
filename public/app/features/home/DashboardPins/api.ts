import { getBackendSrv } from '@grafana/runtime';

import {
  type CreateDashboardPinPayload,
  type DashboardPin,
  type ReorderDashboardPinsPayload,
  type UpdateDashboardPinPayload,
} from './types';

interface CreateDashboardPinResponse {
  pin: DashboardPin;
}

interface ListDashboardPinsResponse {
  pins: DashboardPin[];
}

interface UpdateDashboardPinResponse {
  pin: DashboardPin;
}

export async function createDashboardPin(payload: CreateDashboardPinPayload): Promise<DashboardPin> {
  const response = await getBackendSrv().post<CreateDashboardPinResponse>('/api/home/dashboard-pins', payload);
  return response.pin;
}

export async function listDashboardPins(): Promise<DashboardPin[]> {
  const response = await getBackendSrv().get<ListDashboardPinsResponse>('/api/home/dashboard-pins');
  return response.pins ?? [];
}

export async function updateDashboardPin(uid: string, payload: UpdateDashboardPinPayload): Promise<DashboardPin> {
  const response = await getBackendSrv().patch<UpdateDashboardPinResponse>(`/api/home/dashboard-pins/${uid}`, payload);
  return response.pin;
}

export async function reorderDashboardPins(payload: ReorderDashboardPinsPayload): Promise<void> {
  await getBackendSrv().put('/api/home/dashboard-pins/reorder', payload);
}

export async function deleteDashboardPin(uid: string): Promise<void> {
  await getBackendSrv().delete(`/api/home/dashboard-pins/${uid}`);
}
