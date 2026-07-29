import { css } from '@emotion/css';
import { DragDropContext, Draggable, Droppable, type DropResult } from '@hello-pangea/dnd';

import { type GrafanaTheme2 } from '@grafana/data';
import { t, Trans } from '@grafana/i18n';
import { Button, EmptyState, Icon, IconButton, Input, Link, Stack, Text, useStyles2 } from '@grafana/ui';
import PageLoader from 'app/core/components/PageLoader/PageLoader';
import { type LocationInfo } from 'app/features/search/service/types';

import { pinClicked, pinNoteUpdated, pinReordered, unpinClicked } from '../analytics/main';

import { type PinnedDashboardView } from '../pinned/types';

interface Props {
  dashboards: PinnedDashboardView[];
  loading: boolean;
  error: Error | undefined;
  retry: () => void;
  foldersByUid: Record<string, LocationInfo>;
  onUnpin: (uid: string) => Promise<void>;
  onReorder: (dashboardUids: string[]) => Promise<void>;
  onUpdateNote: (uid: string, note: string) => Promise<void>;
}

export function PinnedDashboardsShelf({
  dashboards,
  loading,
  error,
  retry,
  foldersByUid,
  onUnpin,
  onReorder,
  onUpdateNote,
}: Props) {
  const styles = useStyles2(getStyles);

  if (loading) {
    return <PageLoader text={t('home.pinned-shelf.loading', 'Loading pinned dashboards...')} />;
  }

  if (error) {
    return (
      <Stack direction="column" gap={1}>
        <Text variant="h5">{t('home.pinned-shelf.title', 'Pinned')}</Text>
        <Text color="error">{t('home.pinned-shelf.error-title', 'Could not load pinned dashboards')}</Text>
        <Button size="sm" variant="secondary" onClick={retry}>
          <Trans i18nKey="home.pinned-shelf.retry">Retry</Trans>
        </Button>
      </Stack>
    );
  }

  const handleDragEnd = async (result: DropResult) => {
    if (!result.destination || result.source.index === result.destination.index) {
      return;
    }

    const next = [...dashboards];
    const [moved] = next.splice(result.source.index, 1);
    next.splice(result.destination.index, 0, moved);

    pinReordered({ dashboard_count: next.length });
    await onReorder(next.map((dashboard) => dashboard.uid));
  };

  return (
    <Stack direction="column" gap={1}>
      <Text variant="h5">{t('home.pinned-shelf.title', 'Pinned')}</Text>

      {dashboards.length === 0 ? (
        <EmptyState
          hideImage
          variant="completed"
          message={t('home.pinned-shelf.empty', 'Pinned dashboards will appear here.')}
        >
          <Trans i18nKey="home.pinned-shelf.empty-description">
            Pin dashboards from the Recent tab to keep your go-to views on Home.
          </Trans>
        </EmptyState>
      ) : (
        <DragDropContext onDragEnd={handleDragEnd}>
          <Droppable droppableId="pinned-dashboards" direction="vertical">
            {(provided) => (
              <ul className={styles.list} ref={provided.innerRef} {...provided.droppableProps}>
                {dashboards.map((dashboard, index) => (
                  <Draggable key={dashboard.uid} draggableId={dashboard.uid} index={index}>
                    {(dragProvided) => (
                      <li
                        className={styles.item}
                        ref={dragProvided.innerRef}
                        {...dragProvided.draggableProps}
                        data-testid={`pinned-dashboard-${dashboard.uid}`}
                      >
                        <div className={styles.dragHandle} {...dragProvided.dragHandleProps}>
                          <Icon name="draggabledots" />
                        </div>
                        <div className={styles.content}>
                          <Link href={dashboard.url}>
                            <Text element="p">{dashboard.name}</Text>
                          </Link>
                          {foldersByUid[dashboard.location] && (
                            <Text color="secondary" variant="bodySmall" element="p">
                              {foldersByUid[dashboard.location].name}
                            </Text>
                          )}
                          <Input
                            className={styles.noteInput}
                            placeholder={t('home.pinned-shelf.note-placeholder', 'Add a note')}
                            defaultValue={dashboard.note ?? ''}
                            onBlur={(event) => {
                              const note = event.currentTarget.value.trim();
                              if (note !== (dashboard.note ?? '')) {
                                pinNoteUpdated({ uid: dashboard.uid });
                                void onUpdateNote(dashboard.uid, note);
                              }
                            }}
                          />
                        </div>
                        <IconButton
                          name="times"
                          tooltip={t('home.pinned-shelf.unpin', 'Unpin dashboard')}
                          onClick={() => {
                            unpinClicked({ uid: dashboard.uid });
                            void onUnpin(dashboard.uid);
                          }}
                        />
                      </li>
                    )}
                  </Draggable>
                ))}
                {provided.placeholder}
              </ul>
            )}
          </Droppable>
        </DragDropContext>
      )}
    </Stack>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  list: css({
    listStyle: 'none',
    padding: 0,
    margin: 0,
  }),
  item: css({
    display: 'flex',
    alignItems: 'flex-start',
    gap: theme.spacing(1),
    padding: theme.spacing(1),
    border: `1px solid ${theme.colors.border.weak}`,
    borderRadius: theme.shape.radius.default,
    marginBottom: theme.spacing(0.5),
    background: theme.colors.background.primary,
  }),
  dragHandle: css({
    cursor: 'grab',
    paddingTop: theme.spacing(0.5),
  }),
  content: css({
    flex: 1,
    minWidth: 0,
  }),
  noteInput: css({
    marginTop: theme.spacing(0.5),
  }),
});
