import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { render } from '../../../../test/test-utils';
import { makeExplorePaneState } from '../state/utils';

import { ExploreBookmarksModal } from './ExploreBookmarksModal';
import * as exploreBookmarkApi from './exploreBookmarkApi';

jest.mock('./exploreBookmarkApi');

jest.mock('../../plugins/datasource_srv', () => ({
  getDatasourceSrv: () => ({
    getInstanceSettings: () => ({ name: 'Prometheus' }),
  }),
}));

const mockBookmark = {
  uid: 'abc123',
  name: 'CPU usage',
  datasourceUid: 'prometheus',
  queries: [{ refId: 'A', expr: 'up' }],
  timeFrom: 'now-6h',
  timeTo: 'now',
  createdAt: 1,
  updatedAt: 1,
};

describe('ExploreBookmarksModal', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(exploreBookmarkApi, 'listExploreBookmarks').mockResolvedValue([mockBookmark]);
    jest.spyOn(exploreBookmarkApi, 'createExploreBookmark').mockResolvedValue({
      ...mockBookmark,
      uid: 'new-bookmark',
      name: 'My bookmark',
    });
    jest.spyOn(exploreBookmarkApi, 'deleteExploreBookmark').mockResolvedValue();
  });

  const renderModal = () =>
    render(<ExploreBookmarksModal exploreId="left" isOpen onClose={jest.fn()} />, {
      preloadedState: {
        explore: {
          panes: {
            left: makeExplorePaneState({
              queries: [{ refId: 'A', expr: 'up' }],
              range: {
                from: new Date(),
                to: new Date(),
                raw: { from: 'now-6h', to: 'now' },
              },
              datasourceInstance: {
                uid: 'prometheus',
                name: 'Prometheus',
              } as never,
            }),
          },
        },
      },
    });

  it('renders saved bookmarks', async () => {
    renderModal();
    expect(await screen.findByText('CPU usage')).toBeInTheDocument();
  });

  it('saves the current query as a bookmark', async () => {
    const user = userEvent.setup();
    renderModal();

    await user.type(screen.getByPlaceholderText(/CPU usage last 6 hours/i), 'My bookmark');
    await user.click(screen.getByRole('button', { name: /Save current/i }));

    await waitFor(() => {
      expect(exploreBookmarkApi.createExploreBookmark).toHaveBeenCalledWith({
        name: 'My bookmark',
        datasourceUid: 'prometheus',
        queries: [{ refId: 'A', expr: 'up' }],
        timeFrom: 'now-6h',
        timeTo: 'now',
      });
    });
  });
});
