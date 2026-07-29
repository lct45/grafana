import { reportInteraction } from '@grafana/runtime';
import { t } from '@grafana/i18n';
import { Card, Icon, IconButton, Link, Stack, Text, useStyles2 } from '@grafana/ui';
import { type LocationInfo } from 'app/features/search/service/types';
import { StarToolbarButton } from 'app/features/stars/StarToolbarButton';

import { type Dashboard } from './DashList';
import { getStyles } from './styles';

interface Props {
  dashboard: Dashboard;
  url: string;
  showFolderNames: boolean;
  locationInfo?: LocationInfo;
  layoutMode: 'list' | 'card';
  source: string; // for rudderstack analytics to track which page DashListItem click from
  order?: number; // for rudderstack analytics to track position in cards
  onStarChange?: (id: string, isStarred: boolean) => void;
  isPinned?: boolean;
  onPinToggle?: (shouldPin: boolean) => void;
}
export function DashListItem({
  dashboard,
  url,
  showFolderNames,
  locationInfo,
  layoutMode,
  order,
  onStarChange,
  source,
  isPinned,
  onPinToggle,
}: Props) {
  const css = useStyles2(getStyles);

  const onCardLinkClick = () => {
    reportInteraction('grafana_browse_dashboards_page_click_list_item', {
      itemKind: dashboard.kind,
      source,
      uid: dashboard.uid,
      cardOrder: order,
    });
  };

  return (
    <>
      {layoutMode === 'list' ? (
        <div className={css.dashlistLink}>
          <Link href={url} onClick={onCardLinkClick}>
            <Text element="p">{dashboard.name}</Text>
            {showFolderNames && locationInfo && (
              <Text color="secondary" variant="bodySmall" element="p">
                {locationInfo?.name}
              </Text>
            )}
          </Link>
          <StarToolbarButton
            title={dashboard.name}
            group="dashboard.grafana.app"
            kind="Dashboard"
            id={dashboard.uid}
            onStarChange={onStarChange}
          />
          {onPinToggle && (
            <IconButton
              name={isPinned ? 'pin' : 'gf-pin'}
              tooltip={
                isPinned
                  ? t('home.pinned-shelf.unpin', 'Unpin dashboard')
                  : t('home.pinned-shelf.pin', 'Pin dashboard')
              }
              aria-label={
                isPinned
                  ? t('home.pinned-shelf.unpin', 'Unpin dashboard')
                  : t('home.pinned-shelf.pin', 'Pin dashboard')
              }
              onClick={() => onPinToggle(!isPinned)}
            />
          )}
        </div>
      ) : (
        <Card noMargin className={css.dashlistCardContainer}>
          <Stack justifyContent="space-between" alignItems="start" height="100%">
            <Link
              className={css.dashlistCard}
              href={url}
              aria-label={dashboard.name}
              title={dashboard.name}
              onClick={onCardLinkClick}
            >
              <div className={css.dashlistCardLink}>{dashboard.name}</div>

              {showFolderNames && locationInfo && (
                <Stack alignItems="start" direction="row" gap={0.5}>
                  <Icon name="folder" size="sm" className={css.dashlistCardIcon} aria-hidden="true" />
                  <div className={css.dashlistCardFolder}>
                    <Text
                      color="secondary"
                      variant="bodySmall"
                      element="p"
                      aria-label={locationInfo?.name}
                      title={locationInfo?.name}
                    >
                      {locationInfo?.name}
                    </Text>
                  </div>
                </Stack>
              )}
            </Link>

            <StarToolbarButton
              title={dashboard.name}
              group="dashboard.grafana.app"
              kind="Dashboard"
              id={dashboard.uid}
              onStarChange={onStarChange}
            />
          </Stack>
        </Card>
      )}
    </>
  );
}
