import { useState } from 'react';
import { Modal, Button, App, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import { uploadDiagnosis as defaultUpload } from '../api/checkins';
import { extractApiError } from '../utils/apiError';

interface DiagnosisUploadModalProps {
  open: boolean;
  schedulePatientId: number;
  onClose: () => void;
  onUploaded: () => void;
  /** Injected for tests; defaults to the real API call. */
  upload?: (spID: number, files: File[]) => Promise<unknown>;
}

const MAX_PHOTOS = 3;

/**
 * DiagnosisUploadModal lets a translator (or admin via the same shape) attach
 * up to 3 diagnosis photos to one SchedulePatient. Upload uses multipart form.
 */
export default function DiagnosisUploadModal({
  open,
  schedulePatientId,
  onClose,
  onUploaded,
  upload = defaultUpload,
}: DiagnosisUploadModalProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [files, setFiles] = useState<File[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const handleSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const list = Array.from(e.target.files ?? []);
    if (list.length > MAX_PHOTOS) {
      // Auto-cap: keep the first 3, silently drop the rest. Warn so the user
      // knows their later picks were ignored rather than letting them think
      // everything was uploaded.
      void message.warning(t('diagnosis.tooManyFiles'));
      setFiles(list.slice(0, MAX_PHOTOS));
      return;
    }
    setFiles(list);
  };

  const handleSubmit = async () => {
    if (files.length === 0) return;
    setSubmitting(true);
    try {
      await upload(schedulePatientId, files);
      void message.success(t('diagnosis.uploaded'));
      setFiles([]);
      onUploaded();
      onClose();
    } catch (err: unknown) {
      // Show the backend-translated message (set by axios interceptor) when
      // available; fall back to the generic "Failed" only as last resort.
      void message.error(extractApiError(err) || t('common.failed'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={t('diagnosis.upload')}
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnClose
    >
      <Space direction="vertical" style={{ width: '100%' }}>
        <input
          type="file"
          accept="image/*"
          multiple
          onChange={handleSelect}
        />
        {files.length > 0 && (
          <div style={{ fontSize: 12, color: '#666' }}>
            {files.map((f) => f.name).join(', ')}
          </div>
        )}
        <Button
          type="primary"
          block
          disabled={files.length === 0 || files.length > MAX_PHOTOS}
          loading={submitting}
          onClick={handleSubmit}
        >
          {submitting ? t('diagnosis.uploading') : t('diagnosis.submit')}
        </Button>
      </Space>
    </Modal>
  );
}
