import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { createTheme, type ThemeRegistryItem } from '@grafana/data';

import { ThemeCard } from './ThemeCard';

describe('ThemeCard', () => {
  let user: ReturnType<typeof userEvent.setup>;

  beforeEach(() => {
    user = userEvent.setup();
  });

  const mockTheme: ThemeRegistryItem = {
    id: 'dark',
    name: 'Dark',
    build: createTheme,
  };

  it('should only call onSelect once when clicking the radio button dot', async () => {
    const onSelectMock = jest.fn();

    render(<ThemeCard themeOption={mockTheme} onSelect={onSelectMock} isSelected={false} />);

    // Find the radio button input element
    const radioButtonInput = screen.getByRole('radio');

    // Click the radio button
    await user.click(radioButtonInput);

    // Check that onSelect was called only once
    expect(onSelectMock).toHaveBeenCalledTimes(1);
  });

  it('renders the Cotton Candy theme label', () => {
    const cottonCandyTheme: ThemeRegistryItem = {
      id: 'cottoncandy',
      name: 'Cotton Candy',
      build: () => createTheme({ colors: { mode: 'light' } }),
    };

    render(<ThemeCard themeOption={cottonCandyTheme} onSelect={jest.fn()} isSelected={false} />);

    expect(screen.getByRole('radio', { name: 'Cotton Candy' })).toBeInTheDocument();
  });

  it('does not show an experimental badge for Cotton Candy', () => {
    const cottonCandyTheme: ThemeRegistryItem = {
      id: 'cottoncandy',
      name: 'Cotton Candy',
      build: () => createTheme({ colors: { mode: 'light' } }),
    };

    render(<ThemeCard themeOption={cottonCandyTheme} onSelect={jest.fn()} isSelected={false} />);

    expect(screen.queryByText(/experimental/i)).not.toBeInTheDocument();
  });
});
