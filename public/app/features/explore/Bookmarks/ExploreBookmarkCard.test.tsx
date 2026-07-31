import { screen } from '@testing-library/react';

import { dateTime } from '@grafana/data';
import { initialUserState } from 'app/features/profile/state/reducers';

import { render } from '../../../../test/test-utils';

import { ExploreBookmarkCard } from './ExploreBookmarkCard';
import { type ExploreBookmark } from './types';

jest.mock('../../plugins/datasource_srv', () => ({
  getDatasourceSrv: () => ({
    getInstanceSettings: () => ({ name: 'Prometheus' }),
  }),
}));

const baseBookmark: ExploreBookmark = {
  uid: 'abc123',
  name: 'CPU usage',
  datasourceUid: 'prometheus',
  queries: [{ refId: 'A' }],
  timeFrom: 'now-6h',
  timeTo: 'now',
  createdAt: 1,
  updatedAt: 1,
};

const renderCard = (bookmark: ExploreBookmark) =>
  render(<ExploreBookmarkCard bookmark={bookmark} onOpen={jest.fn()} onDelete={jest.fn()} />, {
    preloadedState: {
      user: { ...initialUserState, timeZone: 'utc' },
    },
  });

describe('ExploreBookmarkCard', () => {
  it('renders absolute ranges stored as epoch millis as readable datetimes', () => {
    const from = dateTime('2024-01-15T10:00:00.000Z');
    const to = dateTime('2024-01-15T11:00:00.000Z');
    renderCard({ ...baseBookmark, timeFrom: String(from.valueOf()), timeTo: String(to.valueOf()) });

    expect(screen.getByText(/Prometheus · 2024-01-15 10:00:00 to 2024-01-15 11:00:00/)).toBeInTheDocument();
    expect(screen.queryByText(new RegExp(String(from.valueOf())))).not.toBeInTheDocument();
  });

  it('renders relative ranges with their quick range name', () => {
    renderCard(baseBookmark);

    expect(screen.getByText(/Prometheus · Last 6 hours/)).toBeInTheDocument();
  });
});
