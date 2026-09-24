import { readFileSync } from 'node:fs';
import path from 'node:path';

import { render, screen } from 'test/test-utils';

import { Branding } from './Branding';

describe('Branding logos', () => {
  it('uses a smiley face for the Grafana icon', () => {
    const svg = readFileSync(path.resolve(__dirname, '../../../../img/grafana_icon.svg'), 'utf8');

    expect(svg.match(/<circle/g)?.length).toBeGreaterThanOrEqual(3);
    expect(svg).toContain('stroke-linecap:round');
    expect(svg).toContain('stop-color:#F05A28');
  });

  it('renders the login logo as the Grafana icon', () => {
    render(<Branding.LoginLogo />);

    const logo = screen.getByRole('img', { name: 'Grafana' });
    expect(logo).toBeInTheDocument();
    expect(logo).toHaveAttribute('src', '__DEFAULT_MOCK_IMAGE_CONTENT__');
  });

  it('renders the menu logo as the Grafana icon', () => {
    render(<Branding.MenuLogo />);

    const logo = screen.getByRole('img', { name: 'Grafana' });
    expect(logo).toBeInTheDocument();
    expect(logo).toHaveAttribute('src', '__DEFAULT_MOCK_IMAGE_CONTENT__');
  });
});
