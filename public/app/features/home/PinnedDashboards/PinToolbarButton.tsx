import { useMemo } from 'react';

import { t } from '@grafana/i18n';
import { Icon, ToolbarButton } from '@grafana/ui';
import { contextSrv } from 'app/core/services/context_srv';

import { useDashboardPins } from './useDashboardPins';

interface Props {
  dashboardUid: string;
  title: string;
}

export function PinToolbarButton({ dashboardUid, title }: Props) {
  const { isPinned, isLoading, pinDashboard, unpinDashboard } = useDashboardPins();

  const pinned = isPinned(dashboardUid);

  const tooltips = useMemo(
    () => ({
      pin: t('home.pinned-dashboards.toolbar-pin', 'Pin to Home'),
      pinWithTitle: t('home.pinned-dashboards.toolbar-pin-with-title', 'Pin "{{title}}" to Home', { title }),
      unpin: t('home.pinned-dashboards.toolbar-unpin', 'Unpin from Home'),
      unpinWithTitle: t('home.pinned-dashboards.toolbar-unpin-with-title', 'Unpin "{{title}}" from Home', { title }),
    }),
    [title]
  );

  if (!contextSrv.isSignedIn) {
    return null;
  }

  const handleToggle = async () => {
    if (pinned) {
      await unpinDashboard(dashboardUid);
    } else {
      await pinDashboard(dashboardUid);
    }
  };

  const tooltipAndLabel = pinned
    ? { tooltip: tooltips.unpin, label: isLoading ? undefined : tooltips.unpinWithTitle }
    : { tooltip: tooltips.pin, label: isLoading ? undefined : tooltips.pinWithTitle };

  const icon = <Icon name="gf-pin" size="lg" type={pinned ? 'mono' : 'default'} key={`${isLoading}-${pinned}`} />;

  return (
    <ToolbarButton
      disabled={isLoading}
      tooltip={tooltipAndLabel.tooltip}
      aria-label={tooltipAndLabel.label}
      aria-pressed={pinned}
      icon={icon}
      data-testid="dashboard-pin-button"
      onClick={handleToggle}
    />
  );
}
