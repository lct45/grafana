import { css } from '@emotion/css';
import { useState } from 'react';

import { rangeUtil, type GrafanaTheme2 } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { Button, ConfirmModal, IconButton, useStyles2 } from '@grafana/ui';
import { createQueryText } from 'app/core/utils/richHistory';
import { getTimeZone } from 'app/features/profile/state/selectors';
import { useSelector } from 'app/types/store';

import { getDatasourceSrv } from '../../plugins/datasource_srv';
import { fromURLRange } from '../state/utils';

import { type ExploreBookmark } from './types';

interface Props {
  bookmark: ExploreBookmark;
  onOpen: (bookmark: ExploreBookmark) => void;
  onDelete: (uid: string) => void;
}

const getStyles = (theme: GrafanaTheme2) => ({
  card: css({
    border: `1px solid ${theme.colors.border.weak}`,
    borderRadius: theme.shape.radius.default,
    padding: theme.spacing(1.5),
    marginBottom: theme.spacing(1),
    backgroundColor: theme.colors.background.secondary,
  }),
  header: css({
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    gap: theme.spacing(1),
    marginBottom: theme.spacing(1),
  }),
  title: css({
    fontWeight: theme.typography.fontWeightMedium,
    marginBottom: theme.spacing(0.5),
  }),
  meta: css({
    color: theme.colors.text.secondary,
    fontSize: theme.typography.bodySmall.fontSize,
    marginBottom: theme.spacing(0.5),
  }),
  queryText: css({
    fontFamily: theme.typography.fontFamilyMonospace,
    fontSize: theme.typography.bodySmall.fontSize,
    wordBreak: 'break-all',
    marginBottom: theme.spacing(1),
  }),
  actions: css({
    display: 'flex',
    gap: theme.spacing(1),
    justifyContent: 'flex-end',
  }),
});

export function ExploreBookmarkListItem({ bookmark, onOpen, onDelete }: Props) {
  const styles = useStyles2(getStyles);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const timeZone = useSelector((state) => getTimeZone(state.user));
  const dsApi = getDatasourceSrv().getInstanceSettings(bookmark.datasourceUid);
  const queryPreview = bookmark.queries.map((query) => createQueryText(query)).join('\n');
  const timeRangeLabel = rangeUtil.describeTimeRange(
    fromURLRange({ from: bookmark.timeRange.from, to: bookmark.timeRange.to }),
    timeZone
  );

  return (
    <>
      <div className={styles.card} data-testid={`explore-bookmark-${bookmark.uid}`}>
        <div className={styles.header}>
          <div>
            <div className={styles.title}>{bookmark.name}</div>
            <div className={styles.meta}>
              {dsApi?.name ?? bookmark.datasourceUid} · {timeRangeLabel}
            </div>
          </div>
          <IconButton
            name="trash-alt"
            tooltip={t('explore.bookmarks.delete-tooltip', 'Delete bookmark')}
            onClick={() => setShowDeleteConfirm(true)}
          />
        </div>
        <div className={styles.queryText}>{queryPreview}</div>
        <div className={styles.actions}>
          <Button size="sm" onClick={() => onOpen(bookmark)}>
            <Trans i18nKey="explore.bookmarks.open">Open</Trans>
          </Button>
        </div>
      </div>
      {showDeleteConfirm && (
        <ConfirmModal
          isOpen
          title={t('explore.bookmarks.delete-title', 'Delete bookmark')}
          body={t('explore.bookmarks.delete-confirmation', 'Are you sure you want to delete "{{name}}"?', {
            name: bookmark.name,
          })}
          confirmText={t('explore.bookmarks.delete-confirm', 'Delete')}
          onConfirm={() => {
            onDelete(bookmark.uid);
            setShowDeleteConfirm(false);
          }}
          onDismiss={() => setShowDeleteConfirm(false)}
        />
      )}
    </>
  );
}
