export const MAX_PIN_NOTE_LENGTH = 256;

export interface DashboardPin {
  dashboardUid: string;
  note?: string;
  sortOrder: number;
  createdAt: number;
}

export interface ListDashboardPinsResponse {
  pins: DashboardPin[];
}

export interface CreateDashboardPinPayload {
  dashboardUid: string;
  note?: string;
}

export interface CreateDashboardPinResponse {
  pin: DashboardPin;
}

export interface ReorderDashboardPinsPayload {
  dashboardUids: string[];
}

export interface PatchDashboardPinPayload {
  note: string | null;
}

export interface PatchDashboardPinResponse {
  pin: DashboardPin;
}

export interface PinnedDashboardListItem {
  pin: DashboardPin;
  title: string;
  url: string;
}
