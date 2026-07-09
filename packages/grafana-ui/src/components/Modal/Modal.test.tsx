import { UNSAFE_PortalProvider } from '@react-aria/overlays';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { getPortalContainer, PortalContainer } from '../Portal/Portal';

import { Modal } from './Modal';
import { ModalTabsHeader } from './ModalTabsHeader';

describe('Modal', () => {
  it('renders nothing by default or when isOpen is false', () => {
    render(<Modal title="Some Title" />);

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders correct contents', () => {
    render(
      <Modal title="Some Title" isOpen>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByLabelText('Some Title')).toBeInTheDocument();

    expect(screen.getByTestId('modal-content')).toBeInTheDocument();
  });

  it('pressing escape calls onDismiss correctly', async () => {
    const onDismiss = jest.fn();

    render(
      <Modal title="Some Title" isOpen onDismiss={onDismiss}>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByLabelText('Some Title')).toBeInTheDocument();
    expect(screen.getByTestId('modal-content')).toBeInTheDocument();

    await userEvent.keyboard('{Escape}');

    expect(onDismiss).toHaveBeenCalled();
  });

  it('clicking backdrop calls onDismiss by default', async () => {
    const onDismiss = jest.fn();

    render(
      <Modal title="Some Title" isOpen onDismiss={onDismiss}>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    await userEvent.click(screen.getByRole('presentation'));

    expect(onDismiss).toHaveBeenCalled();
  });

  it('closeOnBackdropClick={false} prevents dismiss on backdrop click', async () => {
    const onDismiss = jest.fn();

    render(
      <Modal title="Some Title" isOpen onDismiss={onDismiss} closeOnBackdropClick={false}>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    await userEvent.click(screen.getByRole('presentation'));

    expect(onDismiss).not.toHaveBeenCalled();
  });

  it('closeOnBackdropClick={false} works independently of closeOnEscape', async () => {
    const onDismiss = jest.fn();

    render(
      <Modal title="Some Title" isOpen onDismiss={onDismiss} closeOnBackdropClick={false}>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    // Backdrop click should not dismiss
    await userEvent.click(screen.getByRole('presentation'));
    expect(onDismiss).not.toHaveBeenCalled();

    // Escape should still dismiss (closeOnEscape defaults to true)
    await userEvent.keyboard('{Escape}');
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it('onClickBackdrop is called when backdrop is clicked', async () => {
    const onClickBackdrop = jest.fn();

    render(
      <Modal title="Some Title" isOpen onClickBackdrop={onClickBackdrop}>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    await userEvent.click(screen.getByRole('presentation'));

    expect(onClickBackdrop).toHaveBeenCalled();
  });

  it('onClickBackdrop suppresses onDismiss when backdrop is clicked', async () => {
    const onDismiss = jest.fn();
    const onClickBackdrop = jest.fn();

    render(
      <Modal title="Some Title" isOpen onDismiss={onDismiss} onClickBackdrop={onClickBackdrop}>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    await userEvent.click(screen.getByRole('presentation'));

    expect(onClickBackdrop).toHaveBeenCalled();
    expect(onDismiss).not.toHaveBeenCalled();
  });

  it('closeOnEscape={false} prevents dismiss on escape key', async () => {
    const onDismiss = jest.fn();

    render(
      <Modal title="Some Title" isOpen onDismiss={onDismiss} closeOnEscape={false}>
        <div data-testid="modal-content">Content</div>
      </Modal>
    );

    await userEvent.keyboard('{Escape}');

    expect(onDismiss).not.toHaveBeenCalled();
  });

  it('has an accessible name from the visible heading when title is a string', () => {
    render(
      <Modal title="Share" isOpen>
        <div>Content</div>
      </Modal>
    );

    const heading = screen.getByRole('heading', { name: 'Share' });
    expect(heading).toHaveAttribute('id');

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-labelledby', heading.getAttribute('id'));
    expect(dialog).not.toHaveAttribute('aria-label');
  });

  it('has an accessible name from a custom title element', () => {
    render(
      <Modal title={<h3>Custom Title</h3>} isOpen>
        <div>Content</div>
      </Modal>
    );

    const heading = screen.getByText('Custom Title');
    const titleWrapper = heading.closest('[id]');
    expect(titleWrapper).toHaveAttribute('id');

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-labelledby', titleWrapper?.getAttribute('id'));
    expect(dialog).not.toHaveAttribute('aria-label');
  });

  it('has an accessible name from ModalTabsHeader custom title', () => {
    render(
      <Modal
        isOpen
        title={
          <ModalTabsHeader
            title="Share Panel"
            tabs={[{ label: 'Link', value: 'link' }]}
            activeTab="link"
            onChangeTab={() => {}}
          />
        }
      >
        <div>Content</div>
      </Modal>
    );

    const heading = screen.getByRole('heading', { name: 'Share Panel' });
    const titleWrapper = heading.closest('[id]');
    expect(titleWrapper).toHaveAttribute('id');

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-labelledby', titleWrapper?.getAttribute('id'));
  });

  it('uses aria-label when provided for custom title elements', () => {
    render(
      <Modal ariaLabel="Share Panel" title={<span>Tabs</span>} isOpen>
        <div>Content</div>
      </Modal>
    );

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-label', 'Share Panel');
    expect(dialog).toHaveAttribute('aria-labelledby');
  });

  // Mirrors the app arrangement (see AppWrapper): UNSAFE_PortalProvider routes react-aria
  // overlays — including the modal and its backdrop — into <PortalContainer />.
  describe('with the app portal container', () => {
    function setup(onDismiss: jest.Mock) {
      const ui = (isOpen: boolean) => (
        <UNSAFE_PortalProvider getContainer={getPortalContainer}>
          <PortalContainer />
          <div data-testid="page-content">Page content</div>
          <Modal title="Some Title" isOpen={isOpen} onDismiss={onDismiss}>
            <div data-testid="modal-content">Content</div>
          </Modal>
        </UNSAFE_PortalProvider>
      );
      const { rerender } = render(ui(false));
      return { openModal: () => rerender(ui(true)) };
    }

    it('pressing an overlay inside the portal container does not dismiss', async () => {
      const onDismiss = jest.fn();
      const { openModal } = setup(onDismiss);

      const overlay = document.createElement('button');
      getPortalContainer().appendChild(overlay);
      openModal();
      await userEvent.click(overlay);

      expect(onDismiss).not.toHaveBeenCalled();
    });

    it('clicking the backdrop still dismisses', async () => {
      const onDismiss = jest.fn();
      const { openModal } = setup(onDismiss);
      openModal();

      const backdrop = screen.getByRole('presentation');
      expect(getPortalContainer().contains(backdrop)).toBe(true);
      await userEvent.click(backdrop);

      expect(onDismiss).toHaveBeenCalled();
    });

    it('pressing page content outside the portal container still dismisses', async () => {
      const onDismiss = jest.fn();
      const { openModal } = setup(onDismiss);
      openModal();

      await userEvent.click(screen.getByTestId('page-content'));

      expect(onDismiss).toHaveBeenCalled();
    });
  });
});
