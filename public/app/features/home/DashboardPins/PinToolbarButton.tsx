import { useMemo, useState } from 'react';

import { t } from '@grafana/i18n';
import { reportInteraction } from '@grafana/runtime';
import { Icon, ToolbarButton } from '@grafana/ui';

import { useDashboardPins } from './useDashboardPins';

const getPinTooltips = (title: string) => ({
  pin: t('home.dashboard-pins.pin', 'Pin to Home'),
  pinWithTitle: t('home.dashboard-pins.pin-with-title', 'Pin "{{title}}" to Home', { title }),
  unpin: t('home.dashboard-pins.unpin', 'Unpin from Home'),
  unpinWithTitle: t('home.dashboard-pins.unpin-with-title', 'Unpin "{{title}}" from Home', { title }),
});

type Props = {
  dashboardUid: string;
  title: string;
  onPinChange?: (dashboardUid: string, isPinned: boolean) => void;
};

export function PinToolbarButton({ dashboardUid, title, onPinChange }: Props) {
  const tooltips = getPinTooltips(title);
  const { isPinned, getPinUid, pinDashboard, unpinDashboard, isLoading } = useDashboardPins();
  const [isUpdating, setIsUpdating] = useState(false);

  const pinned = isPinned(dashboardUid);
  const pinUid = getPinUid(dashboardUid);
  const busy = isLoading || isUpdating;

  const iconProps = useMemo(() => {
    if (busy) {
      return { name: 'spinner', type: 'default' } as const;
    }
    if (pinned) {
      return { name: 'thumb-tack', type: 'mono' } as const;
    }
    return { name: 'thumb-tack', type: 'default' } as const;
  }, [busy, pinned]);

  const tooltipAndLabel = pinned
    ? { tooltip: tooltips.unpin, label: busy ? undefined : tooltips.unpinWithTitle }
    : { tooltip: tooltips.pin, label: busy ? undefined : tooltips.pinWithTitle };

  const handleToggle = async () => {
    setIsUpdating(true);
    try {
      if (pinned && pinUid) {
        await unpinDashboard(pinUid);
        onPinChange?.(dashboardUid, false);
        reportInteraction('grafana_home_dashboard_unpinned', { origin: 'PinToolbarButton' });
      } else {
        await pinDashboard(dashboardUid);
        onPinChange?.(dashboardUid, true);
        reportInteraction('grafana_home_dashboard_pinned', { origin: 'PinToolbarButton' });
      }
    } finally {
      setIsUpdating(false);
    }
  };

  return (
    <ToolbarButton
      disabled={busy}
      tooltip={tooltipAndLabel.tooltip}
      aria-label={tooltipAndLabel.label}
      aria-pressed={pinned}
      icon={<Icon {...iconProps} size="lg" />}
      onClick={handleToggle}
    />
  );
}
