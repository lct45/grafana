import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { dateTime } from '@grafana/data';

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

  const renderModal = (rawRange: { from: string | ReturnType<typeof dateTime>; to: string | ReturnType<typeof dateTime> } = {
    from: 'now-6h',
    to: 'now',
  }) =>
    render(<ExploreBookmarksModal exploreId="left" isOpen onClose={jest.fn()} />, {
      preloadedState: {
        explore: {
          panes: {
            left: makeExplorePaneState({
              queries: [{ refId: 'A', expr: 'up' }],
              range: {
                from: new Date(),
                to: new Date(),
                raw: rawRange,
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

  it('persists absolute DateTime ranges as epoch millis for round-trip restore', async () => {
    const user = userEvent.setup();
    const from = dateTime('2024-01-15T10:00:00.000Z');
    const to = dateTime('2024-01-15T11:00:00.000Z');
    renderModal({ from, to });

    await user.type(screen.getByPlaceholderText(/CPU usage last 6 hours/i), 'Absolute range');
    await user.click(screen.getByRole('button', { name: /Save current/i }));

    await waitFor(() => {
      expect(exploreBookmarkApi.createExploreBookmark).toHaveBeenCalledWith({
        name: 'Absolute range',
        datasourceUid: 'prometheus',
        queries: [{ refId: 'A', expr: 'up' }],
        timeFrom: String(from.valueOf()),
        timeTo: String(to.valueOf()),
      });
    });
  });

  it('opens a bookmark and closes the modal', async () => {
    const user = userEvent.setup();
    const onClose = jest.fn();
    render(<ExploreBookmarksModal exploreId="left" isOpen onClose={onClose} />, {
      preloadedState: {
        explore: {
          panes: {
            left: makeExplorePaneState({
              queries: [{ refId: 'A', expr: 'up' }],
              range: {
                from: new Date(),
                to: new Date(),
                raw: { from: 'now-1h', to: 'now' },
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

    expect(await screen.findByText('CPU usage')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /^Open$/i }));
    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
    });
  });
});
