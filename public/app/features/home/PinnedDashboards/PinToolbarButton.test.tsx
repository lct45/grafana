import userEvent from '@testing-library/user-event';
import { type ReactNode } from 'react';
import { getWrapper, render, screen, waitFor } from 'test/test-utils';

import { contextSrv } from 'app/core/services/context_srv';

import { PinToolbarButton } from './PinToolbarButton';
import * as dashboardPinsApi from './api';
import { DashboardPinsProvider } from './useDashboardPins';

jest.mock('./api');

jest.mock('app/core/copy/appNotification', () => ({
  useAppNotification: () => ({
    success: jest.fn(),
    error: jest.fn(),
  }),
}));

const BaseWrapper = getWrapper({});

function renderButton(props: { dashboardUid: string; title: string }) {
  return render(<PinToolbarButton {...props} />, {
    wrapper: ({ children }: { children: ReactNode }) => (
      <BaseWrapper>
        <DashboardPinsProvider>{children}</DashboardPinsProvider>
      </BaseWrapper>
    ),
  });
}

describe('PinToolbarButton', () => {
  let originalSignedIn: boolean;

  beforeEach(() => {
    originalSignedIn = contextSrv.isSignedIn;
    contextSrv.isSignedIn = true;
    jest.clearAllMocks();
    jest
      .spyOn(dashboardPinsApi, 'listDashboardPins')
      .mockResolvedValue([{ dashboardUid: 'dash-1', sortOrder: 0, createdAt: 1 }]);
    jest.spyOn(dashboardPinsApi, 'createDashboardPin').mockResolvedValue({
      dashboardUid: 'dash-2',
      sortOrder: 1,
      createdAt: 2,
    });
    jest.spyOn(dashboardPinsApi, 'deleteDashboardPin').mockResolvedValue();
  });

  afterEach(() => {
    contextSrv.isSignedIn = originalSignedIn;
  });

  it('shows pinned state from the shared pin list', async () => {
    renderButton({ dashboardUid: 'dash-1', title: 'Production Overview' });

    const button = await screen.findByTestId('dashboard-pin-button');
    expect(button).toHaveAttribute('aria-pressed', 'true');
  });

  it('pins an unpinned dashboard', async () => {
    const user = userEvent.setup();
    renderButton({ dashboardUid: 'dash-2', title: 'On-call Board' });

    const button = await screen.findByTestId('dashboard-pin-button');
    expect(button).toHaveAttribute('aria-pressed', 'false');

    await user.click(button);

    await waitFor(() => {
      expect(dashboardPinsApi.createDashboardPin).toHaveBeenCalledWith({ dashboardUid: 'dash-2' });
      expect(button).toHaveAttribute('aria-pressed', 'true');
    });
  });

  it('unpins a pinned dashboard', async () => {
    const user = userEvent.setup();
    renderButton({ dashboardUid: 'dash-1', title: 'Production Overview' });

    const button = await screen.findByTestId('dashboard-pin-button');
    await user.click(button);

    await waitFor(() => {
      expect(dashboardPinsApi.deleteDashboardPin).toHaveBeenCalledWith('dash-1');
    });
    expect(button).toHaveAttribute('aria-pressed', 'false');
  });
});
