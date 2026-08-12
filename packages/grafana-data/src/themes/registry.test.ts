import { getBuiltInThemes, getThemeById } from './registry';

describe('theme registry', () => {
  it('registers Sandstone as a first-class theme', () => {
    const theme = getThemeById('sandstone');

    expect(theme.name).toBe('Sandstone');
    expect(theme.isLight).toBe(true);
    expect(theme.isDark).toBe(false);
    expect(theme.colors.primary.main).toBe('#C0673E');
    expect(theme.colors.background.canvas).toBe('#F0E8DC');
  });

  it('includes Sandstone alongside core themes in selectable built-in themes', () => {
    const themes = getBuiltInThemes([]);
    const themeIds = themes.map((theme) => theme.id);

    expect(themeIds).toEqual(expect.arrayContaining(['system', 'dark', 'light', 'sandstone']));
    expect(themes.find((theme) => theme.id === 'sandstone')?.isExtra).toBeUndefined();
  });
});
