import harbor from './themeDefinitions/harbor.json';
import { createTheme, NewThemeOptionsSchema } from './createTheme';
import { getThemeById } from './registry';

describe('Harbor theme', () => {
  it('passes theme schema validation', () => {
    const result = NewThemeOptionsSchema.safeParse(harbor);
    expect(result.success).toBe(true);
  });

  it('builds a dark theme with teal accents', () => {
    const theme = createTheme(harbor);

    expect(theme.name).toBe('Harbor');
    expect(theme.isDark).toBe(true);
    expect(theme.colors.mode).toBe('dark');
    expect(theme.colors.primary.main).toBe('#2dd4bf');
    expect(theme.colors.background.page).toBe('#0f1419');
    expect(theme.colors.background.canvas).toBe('#141a22');
  });

  it('is registered and retrievable by id', () => {
    const theme = getThemeById('harbor');

    expect(theme.name).toBe('Harbor');
    expect(theme.colors.primary.main).toBe('#2dd4bf');
  });
});
