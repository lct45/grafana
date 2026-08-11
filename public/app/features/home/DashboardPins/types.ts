export interface DashboardPin {
  uid: string;
  dashboardUid: string;
  sortOrder: number;
  note?: string;
  createdAt: number;
  updatedAt: number;
}

export interface CreateDashboardPinPayload {
  dashboardUid: string;
  note?: string;
}

export interface UpdateDashboardPinPayload {
  note: string;
}

export interface ReorderDashboardPinsPayload {
  uids: string[];
}

export interface PinnedDashboardItem {
  pinUid: string;
  note?: string;
  dashboardUid: string;
  name: string;
  uid: string;
  url: string;
  location: string;
  kind: string;
}
