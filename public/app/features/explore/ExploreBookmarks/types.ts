import { type DataQuery } from '@grafana/schema';

export interface ExploreBookmarkTimeRange {
  from: string;
  to: string;
}

export interface ExploreBookmark {
  uid: string;
  name: string;
  datasourceUid: string;
  queries: DataQuery[];
  timeRange: ExploreBookmarkTimeRange;
  createdAt: number;
}

export interface CreateExploreBookmarkPayload {
  name: string;
  datasourceUid: string;
  queries: DataQuery[];
  timeRange: ExploreBookmarkTimeRange;
}
