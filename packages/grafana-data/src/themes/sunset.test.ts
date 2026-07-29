import sunset from './themeDefinitions/sunset.json';
import { NewThemeOptionsSchema, createTheme } from './createTheme';
import { getThemeById } from './registry';

describe('sunset theme', () => {
  it('parses against NewThemeOptionsSchema', () => {
    const result = NewThemeOptionsSchema.safeParse(sunset);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.id).toBe('sunset');
      expect(result.data.name).toBe('Sunset');
      expect(result.data.colors?.mode).toBe('dark');
    }
  });

  it('builds a dark theme with sunset accents', () => {
    const theme = createTheme(NewThemeOptionsSchema.parse(sunset));
    expect(theme.isDark).toBe(true);
    expect(theme.name).toBe('Sunset');
    expect(theme.colors.mode).toBe('dark');
    expect(theme.colors.primary.main).toBe('#FF7A45');
    expect(theme.colors.accent.main).toBe('#E84A8A');
    expect(theme.colors.background.canvas).toBe('#0F0A12');
    expect(theme.colors.background.page).toBe('#16101A');
  });

  it('is registered and retrievable by id', () => {
    const theme = getThemeById('sunset');
    expect(theme.name).toBe('Sunset');
    expect(theme.isDark).toBe(true);
    expect(theme.colors.primary.main).toBe('#FF7A45');
  });
});
