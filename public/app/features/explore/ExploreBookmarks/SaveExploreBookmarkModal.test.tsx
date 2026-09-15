import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { SaveExploreBookmarkModal } from './SaveExploreBookmarkModal';

describe('SaveExploreBookmarkModal', () => {
  const onClose = jest.fn();
  const onSave = jest.fn().mockResolvedValue(undefined);

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('clears the name when cancelled so reopening starts empty', async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <SaveExploreBookmarkModal isOpen isSaving={false} canSave onClose={onClose} onSave={onSave} />
    );

    const input = screen.getByPlaceholderText('e.g. CPU usage last 6 hours');
    await user.type(input, 'Draft bookmark name');
    expect(input).toHaveValue('Draft bookmark name');

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalled();

    rerender(
      <SaveExploreBookmarkModal isOpen={false} isSaving={false} canSave onClose={onClose} onSave={onSave} />
    );
    rerender(<SaveExploreBookmarkModal isOpen isSaving={false} canSave onClose={onClose} onSave={onSave} />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText('e.g. CPU usage last 6 hours')).toHaveValue('');
    });
  });

  it('clears the name after a successful save', async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <SaveExploreBookmarkModal isOpen isSaving={false} canSave onClose={onClose} onSave={onSave} />
    );

    const input = screen.getByPlaceholderText('e.g. CPU usage last 6 hours');
    await user.type(input, 'Saved bookmark');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith('Saved bookmark');
      expect(onClose).toHaveBeenCalled();
    });

    rerender(
      <SaveExploreBookmarkModal isOpen={false} isSaving={false} canSave onClose={onClose} onSave={onSave} />
    );
    rerender(<SaveExploreBookmarkModal isOpen isSaving={false} canSave onClose={onClose} onSave={onSave} />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText('e.g. CPU usage last 6 hours')).toHaveValue('');
    });
  });
});
