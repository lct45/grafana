import { Trans, t } from '@grafana/i18n';
import { ToolbarButton } from '@grafana/ui';
import { contextSrv } from 'app/core/services/context_srv';

import { useQueriesDrawerContext } from '../QueriesDrawer/QueriesDrawerContext';

import { useExploreBookmarksContext } from './ExploreBookmarksContext';

interface Props {
  exploreId: string;
  iconOnly?: boolean;
}

export function ExploreBookmarksButton({ exploreId, iconOnly = false }: Props) {
  const { drawerOpen, targetExploreId, openDrawer, closeDrawer } = useExploreBookmarksContext();
  const { drawerOpened: queryHistoryOpen, setDrawerOpened: setQueryHistoryOpen } = useQueriesDrawerContext();
  const isActive = drawerOpen && targetExploreId === exploreId;

  if (!contextSrv.isSignedIn) {
    return null;
  }

  return (
    <ToolbarButton
      key="query-bookmarks"
      variant={isActive ? 'active' : 'canvas'}
      aria-label={t('explore.bookmarks.button-aria-label', 'Query bookmarks')}
      icon="bookmark"
      onClick={() => {
        if (isActive) {
          closeDrawer();
          return;
        }
        if (queryHistoryOpen) {
          setQueryHistoryOpen(false);
        }
        openDrawer(exploreId);
      }}
    >
      {!iconOnly && <Trans i18nKey="explore.bookmarks.button">Bookmarks</Trans>}
    </ToolbarButton>
  );
}
