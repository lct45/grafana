import { render, screen } from 'test/test-utils';

import { Branding } from './Branding';

describe('Branding logos', () => {
  it('renders the login logo as a Grafana image', () => {
    render(<Branding.LoginLogo />);

    expect(screen.getByRole('img', { name: 'Grafana' })).toBeInTheDocument();
  });

  it('renders the menu logo as a Grafana image', () => {
    render(<Branding.MenuLogo />);

    expect(screen.getByRole('img', { name: 'Grafana' })).toBeInTheDocument();
  });
});
