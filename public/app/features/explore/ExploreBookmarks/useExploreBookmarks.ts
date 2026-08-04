import { useCallback, useEffect, useState } from 'react';

import { toURLRange } from '@grafana/data';
import { type DataQuery } from '@grafana/schema';
import { createSuccessNotification } from 'app/core/copy/appNotification';
import { clearQueryKeys } from 'app/core/utils/explore';
import { notifyApp } from 'app/core/reducers/appNotification';
import { getExploreItemSelector } from 'app/features/explore/state/selectors';
import { useDispatch, useSelector } from 'app/types/store';

import { createExploreBookmark, deleteExploreBookmark, listExploreBookmarks } from './api';
import { openExploreBookmark } from './openExploreBookmark';
import { type ExploreBookmark } from './types';

export function useExploreBookmarks(exploreId: string) {
  const dispatch = useDispatch();
  const exploreItemSelector = getExploreItemSelector(exploreId);
  const queries = useSelector((state) => exploreItemSelector(state)?.queries ?? []);
  const range = useSelector((state) => exploreItemSelector(state)?.range);
  const datasourceUid = useSelector((state) => exploreItemSelector(state)?.datasourceInstance?.uid);

  const [bookmarks, setBookmarks] = useState<ExploreBookmark[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshBookmarks = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const result = await listExploreBookmarks();
      setBookmarks(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load bookmarks');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshBookmarks();
  }, [refreshBookmarks]);

  const saveBookmark = useCallback(
    async (name: string) => {
      if (!datasourceUid || !range) {
        return;
      }

      setIsSaving(true);
      try {
        const urlRange = toURLRange(range.raw);
        const bookmark = await createExploreBookmark({
          name,
          datasourceUid,
          queries: queries.map(clearQueryKeys) as DataQuery[],
          timeRange: {
            from: String(urlRange.from),
            to: String(urlRange.to),
          },
        });
        setBookmarks((current) => [bookmark, ...current.filter((item) => item.uid !== bookmark.uid)]);
        dispatch(notifyApp(createSuccessNotification(`Bookmark "${bookmark.name}" saved`)));
      } finally {
        setIsSaving(false);
      }
    },
    [datasourceUid, dispatch, queries, range]
  );

  const openBookmark = useCallback(
    async (bookmark: ExploreBookmark) => {
      await dispatch(openExploreBookmark(exploreId, bookmark));
    },
    [dispatch, exploreId]
  );

  const removeBookmark = useCallback(
    async (uid: string) => {
      await deleteExploreBookmark(uid);
      setBookmarks((current) => current.filter((bookmark) => bookmark.uid !== uid));
      dispatch(notifyApp(createSuccessNotification('Bookmark deleted')));
    },
    [dispatch]
  );

  return {
    bookmarks,
    isLoading,
    isSaving,
    error,
    canSave: Boolean(datasourceUid && queries.length > 0 && range),
    refreshBookmarks,
    saveBookmark,
    openBookmark,
    removeBookmark,
  };
}
