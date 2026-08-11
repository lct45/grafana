import { css } from '@emotion/css';
import { DragDropContext, Draggable, Droppable, type DropResult } from '@hello-pangea/dnd';
import { useMemo, useState } from 'react';

import { type GrafanaTheme2 } from '@grafana/data';
import { t, Trans } from '@grafana/i18n';
import { reportInteraction } from '@grafana/runtime';
import { Button, EmptyState, Icon, IconButton, Input, Link, Stack, Text, useStyles2 } from '@grafana/ui';
import PageLoader from 'app/core/components/PageLoader/PageLoader';
import { type LocationInfo } from 'app/features/search/service/types';

import { DashboardTabError } from './DashboardTabError';
import { type PinnedDashboardItem } from '../DashboardPins/types';

interface Props {
  dashboards: PinnedDashboardItem[];
  loading: boolean;
  error: Error | undefined;
  retry: () => void;
  foldersByUid: Record<string, LocationInfo>;
  onReorder: (uids: string[]) => Promise<void>;
  onUnpin: (pinUid: string) => Promise<void>;
  onUpdateNote: (pinUid: string, note: string) => Promise<void>;
}

export function PinnedDashboardsTab({
  dashboards,
  loading,
  error,
  retry,
  foldersByUid,
  onReorder,
  onUnpin,
  onUpdateNote,
}: Props) {
  const styles = useStyles2(getStyles);
  const [editingPinUid, setEditingPinUid] = useState<string>();
  const [draftNote, setDraftNote] = useState('');

  const pinUids = useMemo(() => dashboards.map((dashboard) => dashboard.pinUid), [dashboards]);

  if (loading) {
    return <PageLoader text={t('home.pinned-dashboards-tab.loading', 'Loading pinned dashboards...')} />;
  }

  if (error) {
    return (
      <DashboardTabError
        title={t('home.pinned-dashboards-tab.error-title', 'Could not load pinned dashboards')}
        retry={retry}
      />
    );
  }

  if (dashboards.length === 0) {
    return (
      <Stack grow={1} direction="column" alignItems="center" justifyContent="center">
        <EmptyState
          hideImage
          variant="call-to-action"
          message={t('home.pinned-dashboards-tab.empty', 'Pin dashboards here for quick access from Home.')}
        >
          <Trans i18nKey="home.pinned-dashboards-tab.empty-description">
            Open a dashboard and use the pin button in the toolbar to add it to this shelf.
          </Trans>
        </EmptyState>
      </Stack>
    );
  }

  const onDragEnd = async (result: DropResult) => {
    if (!result.destination || result.source.index === result.destination.index) {
      return;
    }

    const nextUids = [...pinUids];
    const [moved] = nextUids.splice(result.source.index, 1);
    nextUids.splice(result.destination.index, 0, moved);

    reportInteraction('grafana_home_dashboard_pins_reordered');
    await onReorder(nextUids);
  };

  const startEditingNote = (dashboard: PinnedDashboardItem) => {
    setEditingPinUid(dashboard.pinUid);
    setDraftNote(dashboard.note ?? '');
  };

  const saveNote = async (pinUid: string) => {
    await onUpdateNote(pinUid, draftNote.trim());
    setEditingPinUid(undefined);
    setDraftNote('');
  };

  return (
    <DragDropContext onDragEnd={onDragEnd}>
      <Droppable droppableId="pinned-dashboards">
        {(provided) => (
          <ul className={styles.list} ref={provided.innerRef} {...provided.droppableProps}>
            {dashboards.map((dashboard, index) => (
              <Draggable key={dashboard.pinUid} draggableId={dashboard.pinUid} index={index}>
                {(draggableProvided) => (
                  <li
                    ref={draggableProvided.innerRef}
                    {...draggableProvided.draggableProps}
                    className={styles.item}
                    data-testid={`pinned-dashboard-${dashboard.uid}`}
                  >
                    <Stack direction="row" alignItems="center" gap={1}>
                      <span {...draggableProvided.dragHandleProps} className={styles.dragHandle} aria-hidden="true">
                        <Icon name="draggabledots" />
                      </span>
                      <div className={styles.content}>
                        <Link href={dashboard.url}>
                          <Text element="p">{dashboard.name}</Text>
                        </Link>
                        {foldersByUid[dashboard.location] && (
                          <Text color="secondary" variant="bodySmall" element="p">
                            {foldersByUid[dashboard.location].name}
                          </Text>
                        )}
                        {editingPinUid === dashboard.pinUid ? (
                          <Stack direction="row" gap={1} alignItems="center">
                            <Input
                              value={draftNote}
                              placeholder={t('home.pinned-dashboards-tab.note-placeholder', 'Add a short note')}
                              onChange={(event) => setDraftNote(event.currentTarget.value)}
                            />
                            <Button size="sm" onClick={() => saveNote(dashboard.pinUid)}>
                              <Trans i18nKey="home.pinned-dashboards-tab.save-note">Save</Trans>
                            </Button>
                            <Button
                              size="sm"
                              variant="secondary"
                              onClick={() => {
                                setEditingPinUid(undefined);
                                setDraftNote('');
                              }}
                            >
                              <Trans i18nKey="home.pinned-dashboards-tab.cancel-note">Cancel</Trans>
                            </Button>
                          </Stack>
                        ) : (
                          dashboard.note && (
                            <Text color="secondary" variant="bodySmall" element="p">
                              {dashboard.note}
                            </Text>
                          )
                        )}
                      </div>
                      <Stack direction="row" gap={0.5}>
                        <IconButton
                          name="pen"
                          tooltip={t('home.pinned-dashboards-tab.edit-note', 'Edit note')}
                          aria-label={t('home.pinned-dashboards-tab.edit-note', 'Edit note')}
                          onClick={() => startEditingNote(dashboard)}
                        />
                        <IconButton
                          name="times"
                          tooltip={t('home.pinned-dashboards-tab.unpin', 'Unpin dashboard')}
                          aria-label={t('home.pinned-dashboards-tab.unpin', 'Unpin dashboard')}
                          onClick={() => onUnpin(dashboard.pinUid)}
                        />
                      </Stack>
                    </Stack>
                  </li>
                )}
              </Draggable>
            ))}
            {provided.placeholder}
          </ul>
        )}
      </Droppable>
    </DragDropContext>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  list: css({
    listStyle: 'none',
    padding: 0,
    margin: 0,
  }),
  item: css({
    padding: theme.spacing(1, 1.5),
    borderBottom: `1px solid ${theme.colors.border.weak}`,
    background: theme.colors.background.primary,

    '&:last-child': {
      borderBottom: 'none',
    },
  }),
  dragHandle: css({
    display: 'flex',
    alignItems: 'center',
    cursor: 'grab',
    color: theme.colors.text.secondary,
  }),
  content: css({
    flex: 1,
    minWidth: 0,
  }),
});
