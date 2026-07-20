import { css } from '@emotion/css';
import { useState } from 'react';

import { type GrafanaTheme2 } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { Button, Field, Input, Modal, Spinner, useStyles2 } from '@grafana/ui';

import { ExploreBookmarkCard } from './ExploreBookmarkCard';
import { useExploreBookmarks } from './useExploreBookmarks';

interface Props {
  exploreId: string;
  isOpen: boolean;
  onClose: () => void;
}

const getStyles = (theme: GrafanaTheme2) => ({
  saveSection: css({
    display: 'flex',
    gap: theme.spacing(1),
    alignItems: 'flex-end',
    marginBottom: theme.spacing(2),
    paddingBottom: theme.spacing(2),
    borderBottom: `1px solid ${theme.colors.border.weak}`,
  }),
  emptyState: css({
    color: theme.colors.text.secondary,
    textAlign: 'center',
    padding: theme.spacing(4, 2),
  }),
  list: css({
    maxHeight: '50vh',
    overflowY: 'auto',
  }),
});

export function ExploreBookmarksModal({ exploreId, isOpen, onClose }: Props) {
  const styles = useStyles2(getStyles);
  const [name, setName] = useState('');
  const { bookmarks, isLoading, isSaving, canSave, saveBookmark, openBookmark, removeBookmark } =
    useExploreBookmarks(exploreId);

  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      return;
    }
    await saveBookmark(trimmedName);
    setName('');
  };

  const handleOpen = async (bookmark: Parameters<typeof openBookmark>[0]) => {
    await openBookmark(bookmark);
    onClose();
  };

  return (
    <Modal
      title={t('explore.bookmarks.modal-title', 'Query bookmarks')}
      isOpen={isOpen}
      onDismiss={onClose}
      closeOnBackdropClick
    >
      <div className={styles.saveSection}>
        <Field label={t('explore.bookmarks.name-label', 'Bookmark name')} className={css({ flex: 1, marginBottom: 0 })}>
          <Input
            value={name}
            onChange={(event) => setName(event.currentTarget.value)}
            placeholder={t('explore.bookmarks.name-placeholder', 'e.g. CPU usage last 6 hours')}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                handleSave();
              }
            }}
          />
        </Field>
        <Button onClick={handleSave} disabled={!canSave || !name.trim() || isSaving}>
          {isSaving ? <Spinner inline /> : <Trans i18nKey="explore.bookmarks.save">Save current</Trans>}
        </Button>
      </div>

      {isLoading ? (
        <div className={styles.emptyState}>
          <Spinner />
        </div>
      ) : bookmarks.length === 0 ? (
        <div className={styles.emptyState}>
          <Trans i18nKey="explore.bookmarks.empty">No bookmarks yet. Save your current query to get started.</Trans>
        </div>
      ) : (
        <div className={styles.list}>
          {bookmarks.map((bookmark) => (
            <ExploreBookmarkCard
              key={bookmark.uid}
              bookmark={bookmark}
              onOpen={handleOpen}
              onDelete={removeBookmark}
            />
          ))}
        </div>
      )}
    </Modal>
  );
}
