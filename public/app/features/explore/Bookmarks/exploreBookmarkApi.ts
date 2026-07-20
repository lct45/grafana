import { getBackendSrv } from '@grafana/runtime';

import { type CreateExploreBookmarkPayload, type ExploreBookmark } from './types';

interface ExploreBookmarkResponse {
  result: ExploreBookmark;
}

interface ExploreBookmarkListResponse {
  result: ExploreBookmark[];
}

export async function createExploreBookmark(payload: CreateExploreBookmarkPayload): Promise<ExploreBookmark> {
  const response = await getBackendSrv().post<ExploreBookmarkResponse>('/api/explore/bookmarks', payload);
  return response.result;
}

export async function listExploreBookmarks(): Promise<ExploreBookmark[]> {
  const response = await getBackendSrv().get<ExploreBookmarkListResponse>('/api/explore/bookmarks');
  return response.result ?? [];
}

export async function deleteExploreBookmark(uid: string): Promise<void> {
  await getBackendSrv().delete(`/api/explore/bookmarks/${uid}`);
}
