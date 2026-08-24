import { css } from '@emotion/css';
import { type DragEvent, useCallback, useEffect, useMemo, useState } from 'react';

import { type GrafanaTheme2 } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { Button, EmptyState, Icon, IconButton, Input, Link, Stack, Text, ToolbarButton, useStyles2 } from '@grafana/ui';
import PageLoader from 'app/core/components/PageLoader/PageLoader';
import { getGrafanaSearcher } from 'app/features/search/service/searcher';
import { type DashboardQueryResult } from 'app/features/search/service/types';

import { MAX_PIN_NOTE_LENGTH, type PinnedDashboardListItem } from './types';
import { useDashboardPins } from './useDashboardPins';

export function PinnedDashboardsSection() {
  const styles = useStyles2(getSectionStyles);
  const { pins, isLoading, error, refreshPins, reorderPins, unpinDashboard, updatePinNote } = useDashboardPins();
  const [dashboardsByUid, setDashboardsByUid] = useState<Record<string, DashboardQueryResult>>({});
  const [hydrating, setHydrating] = useState(false);
  const [hydrateError, setHydrateError] = useState<string | null>(null);
  const [draggedUid, setDraggedUid] = useState<string | null>(null);

  useEffect(() => {
    if (pins.length === 0) {
      setDashboardsByUid({});
      setHydrateError(null);
      setHydrating(false);
      return;
    }

    let cancelled = false;
    const uids = pins.map((pin) => pin.dashboardUid);

    setHydrating(true);
    setHydrateError(null);

    getGrafanaSearcher()
      .search({ kind: ['dashboard'], uid: uids, limit: uids.length })
      .then((response) => {
        if (cancelled) {
          return;
        }

        const next: Record<string, DashboardQueryResult> = {};
        for (let i = 0; i < response.view.length; i++) {
          const row = response.view.get(i);
          next[row.uid] = row;
        }
        setDashboardsByUid(next);
      })
      .catch((err) => {
        if (cancelled) {
          return;
        }
        setHydrateError(
          err instanceof Error ? err.message : t('home.pinned-dashboards.hydrate-error', 'Failed to load dashboards')
        );
      })
      .finally(() => {
        if (!cancelled) {
          setHydrating(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [pins]);

  const items = useMemo<PinnedDashboardListItem[]>(() => {
    return pins.map((pin) => {
      const dashboard = dashboardsByUid[pin.dashboardUid];
      return {
        pin,
        title: dashboard?.name ?? pin.dashboardUid,
        url: dashboard?.url ?? `/d/${pin.dashboardUid}`,
      };
    });
  }, [dashboardsByUid, pins]);

  const moveItem = useCallback(
    async (fromIndex: number, toIndex: number) => {
      if (fromIndex === toIndex || toIndex < 0 || toIndex >= pins.length) {
        return;
      }

      const nextUids = pins.map((pin) => pin.dashboardUid);
      const [moved] = nextUids.splice(fromIndex, 1);
      nextUids.splice(toIndex, 0, moved);
      await reorderPins(nextUids);
    },
    [pins, reorderPins]
  );

  const handleDrop = useCallback(
    async (targetUid: string) => {
      if (!draggedUid || draggedUid === targetUid) {
        setDraggedUid(null);
        return;
      }

      const fromIndex = pins.findIndex((pin) => pin.dashboardUid === draggedUid);
      const toIndex = pins.findIndex((pin) => pin.dashboardUid === targetUid);
      setDraggedUid(null);

      if (fromIndex === -1 || toIndex === -1) {
        return;
      }

      await moveItem(fromIndex, toIndex);
    },
    [draggedUid, moveItem, pins]
  );

  if (isLoading || (pins.length > 0 && hydrating)) {
    return <PageLoader text={t('home.pinned-dashboards.loading', 'Loading pinned dashboards...')} />;
  }

  if (error) {
    return (
      <Stack direction="column" gap={1}>
        <Text element="h2" variant="h5">
          <Trans i18nKey="home.pinned-dashboards.title">Pinned</Trans>
        </Text>
        <Text color="error">{error}</Text>
        <Button variant="secondary" size="sm" onClick={refreshPins}>
          <Trans i18nKey="home.pinned-dashboards.retry">Try again</Trans>
        </Button>
      </Stack>
    );
  }

  return (
    <Stack direction="column" gap={2} data-testid="pinned-dashboards-section">
      <Text element="h2" variant="h5">
        <Trans i18nKey="home.pinned-dashboards.title">Pinned</Trans>
      </Text>

      {pins.length === 0 ? (
        <EmptyState
          hideImage
          variant="call-to-action"
          message={t('home.pinned-dashboards.empty', 'Pin dashboards from the toolbar to keep them here.')}
        >
          <Trans i18nKey="home.pinned-dashboards.empty-description">
            Open any dashboard and use the pin button next to the star to add it to this list.
          </Trans>
        </EmptyState>
      ) : (
        <>
          {hydrateError && (
            <Text color="warning" variant="bodySmall">
              {hydrateError}
            </Text>
          )}
          <ul className={styles.list} aria-label={t('home.pinned-dashboards.list-label', 'Pinned dashboards')}>
            {items.map((item, index) => (
              <PinnedDashboardItem
                key={item.pin.dashboardUid}
                item={item}
                index={index}
                total={items.length}
                isDragging={draggedUid === item.pin.dashboardUid}
                onDragStart={() => setDraggedUid(item.pin.dashboardUid)}
                onDragEnd={() => setDraggedUid(null)}
                onDrop={() => handleDrop(item.pin.dashboardUid)}
                onMoveUp={() => moveItem(index, index - 1)}
                onMoveDown={() => moveItem(index, index + 1)}
                onUnpin={() => unpinDashboard(item.pin.dashboardUid)}
                onSaveNote={(note) => updatePinNote(item.pin.dashboardUid, note)}
              />
            ))}
          </ul>
        </>
      )}
    </Stack>
  );
}

interface PinnedDashboardItemProps {
  item: PinnedDashboardListItem;
  index: number;
  total: number;
  isDragging: boolean;
  onDragStart: () => void;
  onDragEnd: () => void;
  onDrop: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onUnpin: () => void;
  onSaveNote: (note: string | null) => Promise<void>;
}

function PinnedDashboardItem({
  item,
  index,
  total,
  isDragging,
  onDragStart,
  onDragEnd,
  onDrop,
  onMoveUp,
  onMoveDown,
  onUnpin,
  onSaveNote,
}: PinnedDashboardItemProps) {
  const styles = useStyles2(getItemStyles);
  const [isEditingNote, setIsEditingNote] = useState(false);
  const [draftNote, setDraftNote] = useState(item.pin.note ?? '');
  const [isSavingNote, setIsSavingNote] = useState(false);

  useEffect(() => {
    if (!isEditingNote) {
      setDraftNote(item.pin.note ?? '');
    }
  }, [isEditingNote, item.pin.note]);

  const saveNote = async () => {
    const trimmed = draftNote.trim();
    const nextNote = trimmed.length > 0 ? trimmed.slice(0, MAX_PIN_NOTE_LENGTH) : null;
    const currentNote = item.pin.note ?? null;

    if (nextNote === currentNote) {
      setIsEditingNote(false);
      return;
    }

    setIsSavingNote(true);
    try {
      await onSaveNote(nextNote);
      setIsEditingNote(false);
    } finally {
      setIsSavingNote(false);
    }
  };

  const cancelNoteEdit = () => {
    setDraftNote(item.pin.note ?? '');
    setIsEditingNote(false);
  };

  const handleDragOver = (event: DragEvent<HTMLLIElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  };

  const handleDrop = (event: DragEvent<HTMLLIElement>) => {
    event.preventDefault();
    onDrop();
  };

  return (
    <li
      className={styles.item}
      data-testid={`pinned-dashboard-item-${item.pin.dashboardUid}`}
      draggable
      onDragStart={(event) => {
        event.dataTransfer.effectAllowed = 'move';
        onDragStart();
      }}
      onDragEnd={onDragEnd}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      aria-grabbed={isDragging}
    >
      <Stack alignItems="center" gap={1} grow={1} minWidth={0}>
        <Icon name="draggabledots" className={styles.dragHandle} aria-hidden="true" />
        <Stack direction="column" gap={0.5} grow={1} minWidth={0}>
          <Link href={item.url} className={styles.titleLink}>
            {item.title}
          </Link>
          {isEditingNote ? (
            <Input
              value={draftNote}
              maxLength={MAX_PIN_NOTE_LENGTH}
              disabled={isSavingNote}
              aria-label={t('home.pinned-dashboards.note-input', 'Pin note for {{title}}', { title: item.title })}
              placeholder={t('home.pinned-dashboards.note-placeholder', 'Add a note')}
              onChange={(event) => setDraftNote(event.currentTarget.value)}
              onBlur={saveNote}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  saveNote();
                } else if (event.key === 'Escape') {
                  event.preventDefault();
                  cancelNoteEdit();
                }
              }}
              autoFocus
            />
          ) : (
            <button type="button" className={styles.noteButton} onClick={() => setIsEditingNote(true)}>
              {item.pin.note ? (
                <Text variant="bodySmall" color="secondary">
                  {item.pin.note}
                </Text>
              ) : (
                <Text variant="bodySmall" color="secondary">
                  <Trans i18nKey="home.pinned-dashboards.add-note">Add note</Trans>
                </Text>
              )}
            </button>
          )}
        </Stack>
      </Stack>

      <Stack alignItems="center" gap={0.5} shrink={0}>
        <IconButton
          name="arrow-up"
          tooltip={t('home.pinned-dashboards.move-up', 'Move up')}
          aria-label={t('home.pinned-dashboards.move-up-for', 'Move {{title}} up', { title: item.title })}
          disabled={index === 0}
          onClick={onMoveUp}
        />
        <IconButton
          name="arrow-down"
          tooltip={t('home.pinned-dashboards.move-down', 'Move down')}
          aria-label={t('home.pinned-dashboards.move-down-for', 'Move {{title}} down', { title: item.title })}
          disabled={index === total - 1}
          onClick={onMoveDown}
        />
        <ToolbarButton
          icon={<Icon name="gf-pin" size="md" />}
          tooltip={t('home.pinned-dashboards.unpin', 'Unpin from Home')}
          aria-label={t('home.pinned-dashboards.unpin-for', 'Unpin {{title}} from Home', { title: item.title })}
          onClick={onUnpin}
        />
      </Stack>
    </li>
  );
}

const getSectionStyles = () => ({
  list: css({
    listStyle: 'none',
    padding: 0,
    margin: 0,
  }),
});

const getItemStyles = (theme: GrafanaTheme2) => ({
  item: css({
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: theme.spacing(1, 0),
    borderBottom: `1px solid ${theme.colors.border.weak}`,

    '&:last-child': {
      borderBottom: 'none',
    },
  }),
  dragHandle: css({
    cursor: 'grab',
    color: theme.colors.text.secondary,
  }),
  titleLink: css({
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  }),
  noteButton: css({
    background: 'none',
    border: 'none',
    padding: 0,
    textAlign: 'left',
    cursor: 'text',
    width: '100%',
  }),
});
