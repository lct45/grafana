import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { SaveExploreBookmarkModal } from './SaveExploreBookmarkModal';

describe('SaveExploreBookmarkModal', () => {
  const onClose = jest.fn();
  const onSave = jest.fn().mockResolvedValue(undefined);

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('clears the draft name after cancel so reopen starts empty', async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <SaveExploreBookmarkModal isOpen isSaving={false} canSave onClose={onClose} onSave={onSave} />
    );

    const input = screen.getByPlaceholderText('e.g. CPU usage last 6 hours');
    await user.type(input, 'CPU spike investigation');
    expect(input).toHaveValue('CPU spike investigation');

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalled();

    rerender(<SaveExploreBookmarkModal isOpen isSaving={false} canSave onClose={onClose} onSave={onSave} />);

    expect(screen.getByPlaceholderText('e.g. CPU usage last 6 hours')).toHaveValue('');
  });
});
