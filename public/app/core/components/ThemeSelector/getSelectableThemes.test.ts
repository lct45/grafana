import { getSelectableThemes } from './getSelectableThemes';

describe('getSelectableThemes', () => {
  it('includes Cotton Candy as a built-in theme option', () => {
    const themes = getSelectableThemes();
    const cottonCandy = themes.find((theme) => theme.id === 'cottoncandy');

    expect(cottonCandy).toBeDefined();
    expect(cottonCandy?.name).toBe('Cotton Candy');
    expect(cottonCandy?.isExtra).toBeUndefined();
  });

  it('keeps Light, Dark, and System selectable alongside Cotton Candy', () => {
    const themeIds = getSelectableThemes().map((theme) => theme.id);

    expect(themeIds).toEqual(expect.arrayContaining(['light', 'dark', 'system', 'cottoncandy']));
  });
});
