import { getSelectableThemes } from './getSelectableThemes';

describe('getSelectableThemes', () => {
  it('includes Sandstone as a built-in theme option', () => {
    const themes = getSelectableThemes();
    const sandstone = themes.find((theme) => theme.id === 'sandstone');

    expect(sandstone).toBeDefined();
    expect(sandstone?.name).toBe('Sandstone');
    expect(sandstone?.isExtra).toBeUndefined();
  });

  it('keeps Light, Dark, and System selectable alongside Sandstone', () => {
    const themeIds = getSelectableThemes().map((theme) => theme.id);

    expect(themeIds).toEqual(expect.arrayContaining(['light', 'dark', 'system', 'sandstone']));
  });
});
