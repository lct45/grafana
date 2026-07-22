import ember from './themeDefinitions/ember.json';
import { getThemeById } from './registry';
import { NewThemeOptionsSchema } from './createTheme';

describe('Ember theme', () => {
  it('has a valid theme definition schema', () => {
    const result = NewThemeOptionsSchema.safeParse(ember);

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.id).toBe('ember');
      expect(result.data.name).toBe('Ember');
    }
  });

  it('registers in the theme registry and produces a dark theme', () => {
    const theme = getThemeById('ember');

    expect(theme.isDark).toBe(true);
    expect(theme.colors.mode).toBe('dark');
    expect(theme.colors.text.primary).toBe('#ede8df');
    expect(theme.colors.background.page).toBe('#181510');
    expect(theme.colors.accent.main).toBe('#e8a317');
  });
});
