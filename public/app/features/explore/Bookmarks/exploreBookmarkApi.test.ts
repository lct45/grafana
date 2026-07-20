import { getBackendSrv } from '@grafana/runtime';

import { createExploreBookmark, deleteExploreBookmark, listExploreBookmarks } from './exploreBookmarkApi';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn(),
}));

describe('exploreBookmarkApi', () => {
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
      result: {
        uid: 'abc123',
        name: 'CPU usage',
        datasourceUid: 'prometheus',
        queries: [{ refId: 'A', expr: 'up' }],
        timeFrom: 'now-6h',
        timeTo: 'now',
        createdAt: 1,
        updatedAt: 1,
      },
    });

    const result = await createExploreBookmark({
      name: 'CPU usage',
      datasourceUid: 'prometheus',
      queries: [{ refId: 'A', expr: 'up' }],
      timeFrom: 'now-6h',
      timeTo: 'now',
    });

    expect(backendSrv.post).toHaveBeenCalledWith('/api/explore/bookmarks', {
      name: 'CPU usage',
      datasourceUid: 'prometheus',
      queries: [{ refId: 'A', expr: 'up' }],
      timeFrom: 'now-6h',
      timeTo: 'now',
    });
    expect(result.uid).toBe('abc123');
  });

  it('lists bookmarks', async () => {
    backendSrv.get.mockResolvedValue({
      result: [{ uid: 'abc123', name: 'CPU usage' }],
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
