import { css } from '@emotion/css';
import { useState } from 'react';

import { type GrafanaTheme2 } from '@grafana/data';
import { Trans, t } from '@grafana/i18n';
import { Button, Field, Input, Modal, Spinner, useStyles2 } from '@grafana/ui';

interface Props {
  isOpen: boolean;
  isSaving: boolean;
  canSave: boolean;
  onClose: () => void;
  onSave: (name: string) => Promise<void>;
}

const getStyles = (theme: GrafanaTheme2) => ({
  actions: css({
    display: 'flex',
    justifyContent: 'flex-end',
    gap: theme.spacing(1),
    marginTop: theme.spacing(2),
  }),
});

export function SaveExploreBookmarkModal({ isOpen, isSaving, canSave, onClose, onSave }: Props) {
  const styles = useStyles2(getStyles);
  const [name, setName] = useState('');

  const handleClose = () => {
    // Discard draft name so reopen starts blank after cancel/dismiss.
    setName('');
    onClose();
  };

  const handleSave = async () => {
    const trimmedName = name.trim();
    if (!trimmedName || !canSave || isSaving) {
      return;
    }
    await onSave(trimmedName);
    setName('');
    onClose();
  };

  return (
    <Modal
      title={t('explore.bookmarks.save-modal-title', 'Save bookmark')}
      isOpen={isOpen}
      onDismiss={handleClose}
      closeOnBackdropClick
    >
      <Field label={t('explore.bookmarks.name-label', 'Bookmark name')}>
        <Input
          value={name}
          onChange={(event) => setName(event.currentTarget.value)}
          placeholder={t('explore.bookmarks.name-placeholder', 'e.g. CPU usage last 6 hours')}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              handleSave();
            }
          }}
        />
      </Field>
      <div className={styles.actions}>
        <Button variant="secondary" fill="outline" onClick={handleClose}>
          <Trans i18nKey="explore.bookmarks.cancel">Cancel</Trans>
        </Button>
        <Button onClick={handleSave} disabled={!canSave || !name.trim() || isSaving}>
          {isSaving ? <Spinner inline /> : <Trans i18nKey="explore.bookmarks.save">Save</Trans>}
        </Button>
      </div>
    </Modal>
  );
}
