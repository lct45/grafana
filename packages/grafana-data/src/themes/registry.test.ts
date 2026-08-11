import { getBuiltInThemes, getThemeById } from './registry';

describe('theme registry', () => {
  it('registers Harbor as a built-in dark theme', () => {
    const harbor = getBuiltInThemes([]).find((theme) => theme.id === 'harbor');

    expect(harbor).toBeDefined();
    expect(harbor?.isExtra).toBeUndefined();

    const theme = getThemeById('harbor');
    expect(theme.name).toBe('Harbor');
    expect(theme.colors.mode).toBe('dark');
    expect(theme.colors.background.canvas).toBe('#09141e');
    expect(theme.colors.primary.main).toBe('#2dd4bf');
    expect(theme.colors.accent.main).toBe('#2dd4bf');
  });
});
