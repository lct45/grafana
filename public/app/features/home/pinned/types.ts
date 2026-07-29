export interface PinnedDashboardRecord {
  dashboardUid: string;
  sortOrder: number;
  note?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PinDashboardRequest {
  note?: string;
}

export interface UpdatePinNoteRequest {
  note: string;
}

export interface ReorderPinnedDashboardsRequest {
  dashboardUids: string[];
}

export interface PinnedDashboardView {
  uid: string;
  name: string;
  url: string;
  location: string;
  note?: string;
  sortOrder: number;
}
