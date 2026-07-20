import { type DataQuery } from '@grafana/schema';

export interface ExploreBookmark {
  uid: string;
  name: string;
  datasourceUid: string;
  queries: DataQuery[];
  timeFrom: string;
  timeTo: string;
  createdAt: number;
  updatedAt: number;
}

export interface CreateExploreBookmarkPayload {
  name: string;
  datasourceUid: string;
  queries: DataQuery[];
  timeFrom: string;
  timeTo: string;
}
