import { type ThunkResult } from 'app/types/store';

import { changeDatasource } from '../state/datasource';
import { cancelQueries, runQueries, setQueriesAction } from '../state/query';
import { updateTime } from '../state/time';
import { fromURLRange } from '../state/utils';
import { withUniqueRefIds } from '../utils/queries';

import { type ExploreBookmark } from './types';

/**
 * Restores an Explore pane from a bookmark: datasource first (without importing
 * current queries), then time, then queries, then a single run against the
 * bookmark range.
 */
export function openExploreBookmark(exploreId: string, bookmark: ExploreBookmark): ThunkResult<Promise<void>> {
  return async (dispatch, getState) => {
    const currentUid = getState().explore.panes[exploreId]?.datasourceInstance?.uid;
    if (bookmark.datasourceUid !== currentUid) {
      await dispatch(
        changeDatasource({
          exploreId,
          datasource: bookmark.datasourceUid,
          options: { importQueries: false },
        })
      );
    }

    dispatch(
      updateTime({
        exploreId,
        rawRange: fromURLRange({ from: bookmark.timeRange.from, to: bookmark.timeRange.to }),
      })
    );
    dispatch(setQueriesAction({ exploreId, queries: withUniqueRefIds(bookmark.queries) }));
    await dispatch(cancelQueries(exploreId));
    dispatch(runQueries({ exploreId }));
  };
}
