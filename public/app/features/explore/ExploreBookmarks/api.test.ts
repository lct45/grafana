import { getBackendSrv } from '@grafana/runtime';
import { type DataQuery } from '@grafana/schema';

import { createExploreBookmark, deleteExploreBookmark, listExploreBookmarks } from './api';

const sampleQuery = { refId: 'A', expr: 'up' } as DataQuery;

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn(),
}));

describe('explore bookmarks api', () => {
  const backendSrv = {
    post: jest.fn(),
    get: jest.fn(),
    delete: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    jest.mocked(getBackendSrv).mockReturnValue(backendSrv as unknown as ReturnType<typeof getBackendSrv>);
  });

  it('creates a bookmark', async () => {
    backendSrv.post.mockResolvedValue({
      bookmark: {
        uid: 'abc123',
        name: 'CPU usage',
        datasourceUid: 'prometheus',
        queries: [sampleQuery],
        timeRange: { from: 'now-6h', to: 'now' },
        createdAt: 1,
      },
    });

    const result = await createExploreBookmark({
      name: 'CPU usage',
      datasourceUid: 'prometheus',
      queries: [sampleQuery],
      timeRange: { from: 'now-6h', to: 'now' },
    });

    expect(backendSrv.post).toHaveBeenCalledWith('/api/explore/bookmarks', {
      name: 'CPU usage',
      datasourceUid: 'prometheus',
      queries: [sampleQuery],
      timeRange: { from: 'now-6h', to: 'now' },
    });
    expect(result.uid).toBe('abc123');
  });

  it('lists bookmarks', async () => {
    backendSrv.get.mockResolvedValue({
      bookmarks: [{ uid: 'abc123', name: 'CPU usage' }],
    });

    const result = await listExploreBookmarks();

    expect(backendSrv.get).toHaveBeenCalledWith('/api/explore/bookmarks');
    expect(result).toHaveLength(1);
  });

  it('deletes a bookmark', async () => {
    await deleteExploreBookmark('abc123');
    expect(backendSrv.delete).toHaveBeenCalledWith('/api/explore/bookmarks/abc123');
  });
});
