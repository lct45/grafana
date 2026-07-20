import { useCallback, useEffect, useState } from 'react';

import { type DataQuery } from '@grafana/schema';
import { createSuccessNotification } from 'app/core/copy/appNotification';
import { notifyApp } from 'app/core/reducers/appNotification';
import { changeDatasource } from 'app/features/explore/state/datasource';
import { setQueries, runQueries } from 'app/features/explore/state/query';
import { getExploreItemSelector } from 'app/features/explore/state/selectors';
import { updateTimeRange } from 'app/features/explore/state/time';
import { useDispatch, useSelector } from 'app/types/store';

import { createExploreBookmark, deleteExploreBookmark, listExploreBookmarks } from './exploreBookmarkApi';
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

  const refreshBookmarks = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await listExploreBookmarks();
      setBookmarks(result);
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
        const bookmark = await createExploreBookmark({
          name,
          datasourceUid,
          queries: queries as DataQuery[],
          timeFrom: String(range.raw.from),
          timeTo: String(range.raw.to),
        });
        setBookmarks((current) => [bookmark, ...current.filter((item) => item.uid !== bookmark.uid)]);
        dispatch(
          notifyApp(createSuccessNotification(`Bookmark "${bookmark.name}" saved`))
        );
      } finally {
        setIsSaving(false);
      }
    },
    [datasourceUid, dispatch, queries, range]
  );

  const openBookmark = useCallback(
    async (bookmark: ExploreBookmark) => {
      if (bookmark.datasourceUid !== datasourceUid) {
        await dispatch(changeDatasource({ exploreId, datasource: bookmark.datasourceUid }));
      }
      dispatch(setQueries(exploreId, bookmark.queries));
      dispatch(
        updateTimeRange({
          exploreId,
          rawRange: { from: bookmark.timeFrom, to: bookmark.timeTo },
        })
      );
      dispatch(runQueries({ exploreId }));
    },
    [datasourceUid, dispatch, exploreId]
  );

  const removeBookmark = useCallback(async (uid: string) => {
    await deleteExploreBookmark(uid);
    setBookmarks((current) => current.filter((bookmark) => bookmark.uid !== uid));
    dispatch(notifyApp(createSuccessNotification('Bookmark deleted')));
  }, [dispatch]);

  return {
    bookmarks,
    isLoading,
    isSaving,
    canSave: Boolean(datasourceUid && queries.length > 0 && range),
    refreshBookmarks,
    saveBookmark,
    openBookmark,
    removeBookmark,
  };
}
