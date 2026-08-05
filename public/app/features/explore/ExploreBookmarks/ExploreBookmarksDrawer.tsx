import { css } from '@emotion/css';
import { useState } from 'react';

import { type GrafanaTheme2 } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { Button, Spinner, ToolbarButton, useStyles2 } from '@grafana/ui';

import { ExploreDrawer } from '../ExploreDrawer';

import { ExploreBookmarkListItem } from './ExploreBookmarkList';
import { useExploreBookmarksContext } from './ExploreBookmarksContext';
import { SaveExploreBookmarkModal } from './SaveExploreBookmarkModal';
import { useExploreBookmarks } from './useExploreBookmarks';

const getStyles = (theme: GrafanaTheme2) => ({
  header: css({
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: theme.spacing(1, 2),
    borderBottom: `1px solid ${theme.colors.border.weak}`,
  }),
  title: css({
    fontWeight: theme.typography.fontWeightMedium,
  }),
  body: css({
    padding: theme.spacing(2),
    maxHeight: '50vh',
    overflowY: 'auto',
  }),
  emptyState: css({
    color: theme.colors.text.secondary,
    textAlign: 'center',
    padding: theme.spacing(4, 2),
  }),
});

export function ExploreBookmarksDrawer() {
  const styles = useStyles2(getStyles);
  const { drawerOpen, targetExploreId, closeDrawer } = useExploreBookmarksContext();
  const [saveModalOpen, setSaveModalOpen] = useState(false);

  if (!drawerOpen || !targetExploreId) {
    return null;
  }

  return (
    <ExploreBookmarksDrawerContent
      exploreId={targetExploreId}
      styles={styles}
      saveModalOpen={saveModalOpen}
      setSaveModalOpen={setSaveModalOpen}
      onClose={closeDrawer}
    />
  );
}

function ExploreBookmarksDrawerContent({
  exploreId,
  styles,
  saveModalOpen,
  setSaveModalOpen,
  onClose,
}: {
  exploreId: string;
  styles: ReturnType<typeof getStyles>;
  saveModalOpen: boolean;
  setSaveModalOpen: (open: boolean) => void;
  onClose: () => void;
}) {
  const { bookmarks, isLoading, isSaving, error, canSave, saveBookmark, openBookmark, removeBookmark, refreshBookmarks } =
    useExploreBookmarks(exploreId);

  const handleOpen = async (bookmark: Parameters<typeof openBookmark>[0]) => {
    await openBookmark(bookmark);
    onClose();
  };

  return (
    <>
      <ExploreDrawer>
        <div className={styles.header}>
          <div className={styles.title}>
            <Trans i18nKey="explore.bookmarks.drawer-title">Query bookmarks</Trans>
          </div>
          <div>
            <Button size="sm" variant="primary" onClick={() => setSaveModalOpen(true)} disabled={!canSave}>
              <Trans i18nKey="explore.bookmarks.save-current">Save current query</Trans>
            </Button>
            <ToolbarButton
              icon="times"
              tooltip={t('explore.bookmarks.close', 'Close bookmarks')}
              onClick={onClose}
              aria-label={t('explore.bookmarks.close', 'Close bookmarks')}
            />
          </div>
        </div>
        <div className={styles.body}>
          {isLoading ? (
            <div className={styles.emptyState}>
              <Spinner />
            </div>
          ) : error ? (
            <div className={styles.emptyState}>
              <div>{error}</div>
              <Button size="sm" variant="secondary" onClick={refreshBookmarks}>
                <Trans i18nKey="explore.bookmarks.retry">Retry</Trans>
              </Button>
            </div>
          ) : bookmarks.length === 0 ? (
            <div className={styles.emptyState}>
              <Trans i18nKey="explore.bookmarks.empty">No bookmarks yet. Save your current query to get started.</Trans>
            </div>
          ) : (
            bookmarks.map((bookmark) => (
              <ExploreBookmarkListItem
                key={bookmark.uid}
                bookmark={bookmark}
                onOpen={handleOpen}
                onDelete={removeBookmark}
              />
            ))
          )}
        </div>
      </ExploreDrawer>
      <SaveExploreBookmarkModal
        isOpen={saveModalOpen}
        isSaving={isSaving}
        canSave={canSave}
        onClose={() => setSaveModalOpen(false)}
        onSave={saveBookmark}
      />
    </>
  );
}
