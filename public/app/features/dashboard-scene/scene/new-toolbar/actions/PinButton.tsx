import { PinToolbarButton } from 'app/features/home/PinnedDashboards/PinToolbarButton';

import { type ToolbarActionProps } from '../types';

export const PinButton = ({ dashboard }: ToolbarActionProps) => {
  const { uid, title } = dashboard.useState();
  if (!uid) {
    return null;
  }

  return <PinToolbarButton dashboardUid={uid} title={title} />;
};
