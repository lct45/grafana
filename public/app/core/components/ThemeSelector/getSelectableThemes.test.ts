import { getBuiltInThemes } from '@grafana/data';
import { FlagKeys } from '@grafana/runtime/internal';
import { setTestFlags } from '@grafana/test-utils/unstable';

import { getSelectableThemes } from './getSelectableThemes';

describe('getSelectableThemes', () => {
  afterEach(() => {
    setTestFlags({});
  });

  it('includes Harbor as a selectable extra theme', () => {
    const themes = getSelectableThemes();
    const harbor = themes.find((theme) => theme.id === 'harbor');

    expect(harbor).toBeDefined();
    expect(harbor?.name).toBe('Harbor');
    expect(harbor?.isExtra).toBe(true);
  });

  it('includes built-in themes alongside Harbor', () => {
    const themes = getSelectableThemes();

    expect(themes.some((theme) => theme.id === 'dark')).toBe(true);
    expect(themes.some((theme) => theme.id === 'light')).toBe(true);
    expect(themes.some((theme) => theme.id === 'system')).toBe(true);
    expect(themes.some((theme) => theme.id === 'harbor')).toBe(true);
  });

  it('includes visual refresh themes when the feature flag is enabled', () => {
    setTestFlags({ [FlagKeys.GrafanaVisualDesignRefresh]: true });

    const themes = getSelectableThemes();

    expect(themes.some((theme) => theme.id === 'visual_refresh_dark')).toBe(true);
    expect(themes.some((theme) => theme.id === 'visual_refresh_light')).toBe(true);
  });

  it('matches getBuiltInThemes with the same allowed extras', () => {
    const allowedExtraThemes = [
      'deut_prot_dark',
      'deut_prot_light',
      'tritanopia_dark',
      'tritanopia_light',
      'desertbloom',
      'gildedgrove',
      'harbor',
      'sapphiredusk',
      'tron',
      'gloom',
    ];

    expect(getSelectableThemes().map((theme) => theme.id)).toEqual(
      getBuiltInThemes(allowedExtraThemes).map((theme) => theme.id)
    );
  });
});
