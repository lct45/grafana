import { type ReactNode } from 'react';
import { Provider } from 'react-redux';
import { act, renderHook, waitFor } from '@testing-library/react';

import { dateTime } from '@grafana/data';
import { configureStore } from 'app/store/configureStore';

import { makeExplorePaneState } from '../state/utils';

import * as exploreBookmarkApi from './api';
import { useExploreBookmarks } from './useExploreBookmarks';

jest.mock('./api');

jest.mock('../state/query', () => {
  const actual = jest.requireActual('../state/query');
  return {
    ...actual,
    cancelQueries: jest.fn(() => () => Promise.resolve()),
    runQueries: jest.fn(() => () => undefined),
  };
});

const mockBookmark = {
  uid: 'abc123',
  name: 'CPU usage',
  datasourceUid: 'prometheus',
  queries: [{ refId: 'A', expr: 'up' }],
  timeRange: {
    from: String(dateTime('2024-01-15T10:00:00.000Z').valueOf()),
    to: String(dateTime('2024-01-15T11:00:00.000Z').valueOf()),
  },
  createdAt: 1,
};

function createWrapper(rawRange: {
  from: ReturnType<typeof dateTime> | string;
  to: ReturnType<typeof dateTime> | string;
}) {
  const store = configureStore({
    explore: {
      panes: {
        left: makeExplorePaneState({
          queries: [{ refId: 'A', expr: 'rate(http_requests[5m])' }],
          range: {
            from: dateTime(),
            to: dateTime(),
            raw: rawRange,
          },
          datasourceInstance: {
            uid: 'prometheus',
            name: 'Prometheus',
          } as never,
        }),
      },
    },
  } as never);

  return {
    store,
    wrapper: ({ children }: { children: ReactNode }) => <Provider store={store}>{children}</Provider>,
  };
}

describe('useExploreBookmarks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(exploreBookmarkApi, 'listExploreBookmarks').mockResolvedValue([]);
    jest.spyOn(exploreBookmarkApi, 'createExploreBookmark').mockResolvedValue({
      ...mockBookmark,
      uid: 'new-bookmark',
      name: 'Saved',
    });
    jest.spyOn(exploreBookmarkApi, 'deleteExploreBookmark').mockResolvedValue();
  });

  it('saves absolute DateTime ranges via toURLRange epoch millis', async () => {
    const from = dateTime('2024-01-15T10:00:00.000Z');
    const to = dateTime('2024-01-15T11:00:00.000Z');
    const { wrapper } = createWrapper({ from, to });

    const { result } = renderHook(() => useExploreBookmarks('left'), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    await act(async () => {
      await result.current.saveBookmark('Absolute');
    });

    expect(exploreBookmarkApi.createExploreBookmark).toHaveBeenCalledWith(
      expect.objectContaining({
        timeRange: {
          from: String(from.valueOf()),
          to: String(to.valueOf()),
        },
      })
    );
  });

  it('opens a bookmark by applying parsed time then queries before a single run', async () => {
    const queryState = jest.requireMock('../state/query') as {
      cancelQueries: jest.Mock;
      runQueries: jest.Mock;
    };
    jest.spyOn(exploreBookmarkApi, 'listExploreBookmarks').mockResolvedValue([mockBookmark]);
    const { store, wrapper } = createWrapper({ from: 'now-1h', to: 'now' });

    const { result } = renderHook(() => useExploreBookmarks('left'), { wrapper });

    await waitFor(() => {
      expect(result.current.bookmarks).toHaveLength(1);
    });

    await act(async () => {
      await result.current.openBookmark(mockBookmark);
    });

    expect(queryState.runQueries).toHaveBeenCalledTimes(1);
    expect(queryState.cancelQueries).toHaveBeenCalledWith('left');

    const pane = store.getState().explore.panes.left!;
    expect(pane.queries).toEqual(mockBookmark.queries);
    expect(pane.range.raw.from.valueOf()).toBe(Number(mockBookmark.timeRange.from));
    expect(pane.range.raw.to.valueOf()).toBe(Number(mockBookmark.timeRange.to));
  });

  it('removes a bookmark after delete', async () => {
    jest.spyOn(exploreBookmarkApi, 'listExploreBookmarks').mockResolvedValue([mockBookmark]);
    const { wrapper } = createWrapper({ from: 'now-6h', to: 'now' });
    const { result } = renderHook(() => useExploreBookmarks('left'), { wrapper });

    await waitFor(() => {
      expect(result.current.bookmarks).toHaveLength(1);
    });

    await act(async () => {
      await result.current.removeBookmark(mockBookmark.uid);
    });

    expect(exploreBookmarkApi.deleteExploreBookmark).toHaveBeenCalledWith(mockBookmark.uid);
    expect(result.current.bookmarks).toHaveLength(0);
  });
});
