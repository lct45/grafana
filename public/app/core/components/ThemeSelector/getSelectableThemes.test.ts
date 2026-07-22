import { getThemeById } from '@grafana/data';
import { getFeatureFlagClient, FlagKeys } from '@grafana/runtime/internal';
import { setTestFlags } from '@grafana/test-utils/unstable';

import { getSelectableThemes } from './getSelectableThemes';

describe('getSelectableThemes', () => {
  afterEach(() => {
    setTestFlags({});
  });

  it('includes the Ember theme as a selectable experimental theme', () => {
    const themes = getSelectableThemes();
    const ember = themes.find((theme) => theme.id === 'ember');

    expect(ember).toBeDefined();
    expect(ember?.name).toBe('Ember');
    expect(ember?.isExtra).toBe(true);
  });

  it('builds a warm graphite dark theme with amber accents', () => {
    const theme = getThemeById('ember');

    expect(theme.isDark).toBe(true);
    expect(theme.colors.background.canvas).toBe('#13110c');
    expect(theme.colors.primary.main).toBe('#e8a317');
    expect(theme.colors.warning.main).not.toBe(theme.colors.primary.main);
  });

  it('does not include visual refresh themes unless the feature flag is enabled', () => {
    jest.spyOn(getFeatureFlagClient(), 'getBooleanValue').mockReturnValue(false);

    const themes = getSelectableThemes();

    expect(themes.some((theme) => theme.id === 'visual_refresh_dark')).toBe(false);
    expect(themes.some((theme) => theme.id === 'visual_refresh_light')).toBe(false);
  });

  it('includes visual refresh themes when the feature flag is enabled', () => {
    setTestFlags({ [FlagKeys.GrafanaVisualDesignRefresh]: true });

    const themes = getSelectableThemes();

    expect(themes.some((theme) => theme.id === 'visual_refresh_dark')).toBe(true);
    expect(themes.some((theme) => theme.id === 'visual_refresh_light')).toBe(true);
  });
});
