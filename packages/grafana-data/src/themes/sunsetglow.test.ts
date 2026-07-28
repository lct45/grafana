import { getThemeById } from '@grafana/data';
import { NewThemeOptionsSchema } from '@grafana/data/internal';

import sunsetglow from './themeDefinitions/sunsetglow.json';

describe('sunsetglow theme', () => {
  it('parses the theme definition schema', () => {
    const result = NewThemeOptionsSchema.safeParse(sunsetglow);
    expect(result.success).toBe(true);
  });

  it('builds a dark theme with sunset-inspired colors', () => {
    const theme = getThemeById('sunsetglow');

    expect(theme.name).toBe('Sunset glow');
    expect(theme.isDark).toBe(true);
    expect(theme.colors.primary.main).toBe('#FF6B35');
    expect(theme.colors.background.canvas).toBe('#1A0A2E');
    expect(theme.colors.gradients.brandHorizontal).toContain('#FF6B35');
    expect(theme.colors.gradients.brandHorizontal).toContain('#7B68EE');
  });
});
