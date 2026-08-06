import { getBuiltInThemes, getThemeById } from './registry';

describe('theme registry', () => {
  it('registers Harbor as a first-class built-in theme', () => {
    const themes = getBuiltInThemes([]);
    const harbor = themes.find((theme) => theme.id === 'harbor');

    expect(harbor).toBeDefined();
    expect(harbor?.name).toBe('Harbor');
    expect(harbor?.isExtra).toBeUndefined();
  });

  it('builds the Harbor theme with dark mode and teal accents', () => {
    const harborTheme = getThemeById('harbor');

    expect(harborTheme.isDark).toBe(true);
    expect(harborTheme.colors.primary.main).toBe('#2DD4BF');
    expect(harborTheme.colors.background.canvas).toBe('#1A2332');
    expect(harborTheme.name).toBe('Harbor');
  });
});
