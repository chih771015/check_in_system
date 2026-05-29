import { useState } from 'react';
import { Modal, Button, Input, App, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import { markNoShow as defaultMarkNoShow } from '../api/checkins';
import { extractApiError } from '../utils/apiError';

interface NoShowModalProps {
  open: boolean;
  schedulePatientId: number;
  onClose: () => void;
  onDone: () => void;
  /** Injected for tests; defaults to the real API call. */
  markNoShow?: (spID: number, reason: string) => Promise<unknown>;
}

/**
 * NoShowModal prompts for the no-show reason and updates the SchedulePatient
 * status to no_show. Reason is required (enforced server-side; we also
 * disable submit locally to give immediate feedback).
 */
export default function NoShowModal({
  open,
  schedulePatientId,
  onClose,
  onDone,
  markNoShow = defaultMarkNoShow,
}: NoShowModalProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [reason, setReason] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    const trimmed = reason.trim();
    if (!trimmed) return;
    setSubmitting(true);
    try {
      await markNoShow(schedulePatientId, trimmed);
      void message.success(t('diagnosis.markedNoShow'));
      setReason('');
      onDone();
      onClose();
    } catch (err: unknown) {
      void message.error(extractApiError(err) || t('common.failed'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('diagnosis.noShow')}
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnClose
    >
      <Space direction="vertical" style={{ width: '100%' }}>
        <Input.TextArea
          rows={4}
          placeholder={t('diagnosis.noShowReasonPlaceholder')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        <Button
          type="primary"
          block
          danger
          disabled={!reason.trim()}
          loading={submitting}
          onClick={handleSubmit}
        >
          {t('diagnosis.submit')}
        </Button>
      </Space>
    </Modal>
  );
}
