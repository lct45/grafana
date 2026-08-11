import { type DataQuery } from '@grafana/schema';

import { openExploreBookmark } from './openExploreBookmark';
import { type ExploreBookmark } from './types';

jest.mock('app/core/reducers/appNotification', () => ({
  notifyApp: jest.fn((notification) => ({ type: 'NOTIFY', payload: notification })),
}));

jest.mock('../state/datasource', () => ({
  changeDatasource: jest.fn(() => () => Promise.resolve()),
}));

jest.mock('../state/query', () => ({
  cancelQueries: jest.fn(() => () => Promise.resolve()),
  runQueries: jest.fn(() => () => undefined),
  setQueriesAction: jest.fn((payload) => ({ type: 'SET_QUERIES', payload })),
}));

jest.mock('../state/time', () => ({
  updateTime: jest.fn((payload) => ({ type: 'UPDATE_TIME', payload })),
}));

const bookmark: ExploreBookmark = {
  uid: 'abc123',
  name: 'CPU usage',
  datasourceUid: 'prometheus',
  queries: [{ refId: 'A', expr: 'up' } as DataQuery],
  timeRange: { from: 'now-6h', to: 'now' },
  createdAt: 1,
};

describe('openExploreBookmark', () => {
  it('no-ops when the explore pane no longer exists', async () => {
    const dispatch: jest.Mock = jest.fn((action: unknown) => {
      if (typeof action === 'function') {
        return action(dispatch, () => ({ explore: { panes: {} } }));
      }
      return action;
    });
    const getState = () => ({ explore: { panes: {} } });

    await openExploreBookmark('missing', bookmark)(dispatch, getState as never, undefined);

    const queryState = jest.requireMock('../state/query') as {
      runQueries: jest.Mock;
      setQueriesAction: jest.Mock;
    };
    expect(queryState.runQueries).not.toHaveBeenCalled();
    expect(queryState.setQueriesAction).not.toHaveBeenCalled();
    expect(dispatch).toHaveBeenCalled();
  });
});
