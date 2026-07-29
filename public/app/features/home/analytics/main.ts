import { defineFeatureEvents } from '@grafana/runtime/unstable';

import { type ClearHistoryClicked, type EmptyCtaClicked, type PinClicked, type PinNoteUpdated, type PinReordered, type TabChanged, type UnpinClicked } from './types';

const createHomepageEvent = defineFeatureEvents('grafana', 'homepage');

/** Fired when the user clicks a tab on the homepage. */
export const tabChanged = createHomepageEvent<TabChanged>('tab_changed');

/** Fired when the user clears their recently-viewed dashboard history. */
export const clearHistoryClicked = createHomepageEvent<ClearHistoryClicked>('clear_history_clicked');

/** Fired when the user clicks the empty-state call-to-action on the Recent tab. */
export const emptyCtaClicked = createHomepageEvent<EmptyCtaClicked>('empty_cta_clicked');

/** Fired when the user pins a dashboard from Home. */
export const pinClicked = createHomepageEvent<PinClicked>('pin_clicked');

/** Fired when the user unpins a dashboard from Home. */
export const unpinClicked = createHomepageEvent<UnpinClicked>('unpin_clicked');

/** Fired when the user reorders pinned dashboards. */
export const pinReordered = createHomepageEvent<PinReordered>('pin_reordered');

/** Fired when the user updates a pinned dashboard note. */
export const pinNoteUpdated = createHomepageEvent<PinNoteUpdated>('pin_note_updated');
