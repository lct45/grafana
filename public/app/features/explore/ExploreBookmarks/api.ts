import { getBackendSrv } from '@grafana/runtime';

import { type CreateExploreBookmarkPayload, type ExploreBookmark } from './types';

interface CreateExploreBookmarkResponse {
  bookmark: ExploreBookmark;
}

interface ListExploreBookmarksResponse {
  bookmarks: ExploreBookmark[];
}

export async function createExploreBookmark(payload: CreateExploreBookmarkPayload): Promise<ExploreBookmark> {
  const response = await getBackendSrv().post<CreateExploreBookmarkResponse>('/api/explore/bookmarks', payload);
  return response.bookmark;
}

export async function listExploreBookmarks(): Promise<ExploreBookmark[]> {
  const response = await getBackendSrv().get<ListExploreBookmarksResponse>('/api/explore/bookmarks');
  return response.bookmarks ?? [];
}

export async function deleteExploreBookmark(uid: string): Promise<void> {
  await getBackendSrv().delete(`/api/explore/bookmarks/${uid}`);
}
