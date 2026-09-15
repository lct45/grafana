import { getBuiltInThemes, getThemeById } from './registry';

describe('theme registry', () => {
  it('registers Cotton Candy as a first-class theme', () => {
    const theme = getThemeById('cottoncandy');

    expect(theme.name).toBe('Cotton Candy');
    expect(theme.isLight).toBe(true);
    expect(theme.isDark).toBe(false);
    expect(theme.colors.primary.main).toBe('#FF4FA0');
    expect(theme.colors.background.canvas).toBe('#FFD6EC');
    expect(theme.colors.accent.main).toBe('#4EB5FF');
    expect(theme.typography.fontFamily).toContain('Trebuchet MS');
    expect(theme.shape.radius.md).toBe('16px');
  });

  it('includes Cotton Candy alongside core themes in selectable built-in themes', () => {
    const themes = getBuiltInThemes([]);
    const themeIds = themes.map((theme) => theme.id);

    expect(themeIds).toEqual(expect.arrayContaining(['system', 'dark', 'light', 'cottoncandy']));
    expect(themes.find((theme) => theme.id === 'cottoncandy')?.isExtra).toBeUndefined();
  });
});
