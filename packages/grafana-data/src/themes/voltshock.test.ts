import { getThemeById } from './registry';

describe('voltshock theme', () => {
  it('is registered and builds with punchy neon tokens', () => {
    const theme = getThemeById('voltshock');

    expect(theme.name).toBe('Voltshock');
    expect(theme.isDark).toBe(true);
    expect(theme.colors.primary.main).toBe('#D4FF00');
    expect(theme.colors.accent.main).toBe('#FF2D95');
    expect(theme.colors.background.canvas).toBe('#07040F');
    expect(theme.visualization.palette[0]).toBe('#D4FF00');
  });
});
