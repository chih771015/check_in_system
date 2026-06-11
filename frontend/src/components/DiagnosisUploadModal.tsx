import { useCallback, useEffect, useState } from 'react';
import { Modal, Button, App, Space, Image, Popconfirm, Spin, Typography } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  uploadDiagnosis as defaultUpload,
  listDiagnosisPhotos as defaultListPhotos,
  deleteDiagnosisPhoto as defaultDeletePhoto,
  type DiagnosisPhotoItem,
} from '../api/checkins';
import { extractApiError } from '../utils/apiError';

interface DiagnosisUploadModalProps {
  open: boolean;
  schedulePatientId: number;
  onClose: () => void;
  /** Called after any change (upload or delete) so the parent can refresh. */
  onUploaded: () => void;
  /** Injected for tests; default to the real translator API calls. */
  upload?: (spID: number, files: File[]) => Promise<unknown>;
  listPhotos?: (spID: number) => Promise<DiagnosisPhotoItem[]>;
  deletePhoto?: (photoId: number) => Promise<unknown>;
  /**
   * View-only mode: show the photos but hide all edit affordances (add + delete).
   * Used e.g. for a translator after they have done their leave check-in — they
   * can still review status & photos but can no longer modify them.
   */
  readOnly?: boolean;
}

const MAX_PHOTOS = 3;

/**
 * DiagnosisUploadModal manages up to 3 diagnosis photos for one SchedulePatient.
 *
 * Unlike the original upload-only flow, it now lists already-uploaded photos so
 * the translator can DELETE a photo and ADD more later (e.g. they first picked
 * only one). The modal stays open after each action so several edits can be
 * made in one session; the parent is notified via onUploaded so the slot status
 * (completed / pending) stays in sync.
 */
export default function DiagnosisUploadModal({
  open,
  schedulePatientId,
  onClose,
  onUploaded,
  upload = defaultUpload,
  listPhotos = defaultListPhotos,
  deletePhoto = defaultDeletePhoto,
  readOnly = false,
}: DiagnosisUploadModalProps) {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [existing, setExisting] = useState<DiagnosisPhotoItem[]>([]);
  const [files, setFiles] = useState<File[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const remaining = MAX_PHOTOS - existing.length;

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const list = await listPhotos(schedulePatientId);
      setExisting(list);
    } catch {
      // A missing/empty list is non-fatal; just show no existing photos.
      setExisting([]);
    } finally {
      setLoading(false);
    }
  }, [listPhotos, schedulePatientId]);

  useEffect(() => {
    if (open) {
      setFiles([]);
      void refresh();
    }
  }, [open, refresh]);

  const handleSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const list = Array.from(e.target.files ?? []);
    if (list.length > remaining) {
      // Auto-cap to the number of free slots and warn so the user knows the
      // extras were dropped rather than silently uploaded.
      void message.warning(t('diagnosis.tooManyFiles'));
      setFiles(list.slice(0, Math.max(remaining, 0)));
      return;
    }
    setFiles(list);
  };

  const handleUpload = async () => {
    if (files.length === 0) return;
    setSubmitting(true);
    try {
      await upload(schedulePatientId, files);
      void message.success(t('diagnosis.uploaded'));
      setFiles([]);
      await refresh();
      onUploaded();
    } catch (err: unknown) {
      void message.error(extractApiError(err) || t('common.failed'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (photoId: number) => {
    try {
      await deletePhoto(photoId);
      void message.success(t('diagnosis.photoDeleted'));
      await refresh();
      onUploaded();
    } catch (err: unknown) {
      void message.error(extractApiError(err) || t('common.failed'));
    }
  };

  return (
    <Modal
      title={readOnly ? t('diagnosis.viewPhotos') : t('diagnosis.managePhotos')}
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="done" onClick={onClose}>
          {t('diagnosis.done')}
        </Button>,
      ]}
      destroyOnClose
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        {/* Existing photos with per-photo delete */}
        <div>
          <Typography.Text strong>{t('diagnosis.existingPhotos')}</Typography.Text>
          {loading ? (
            <div style={{ textAlign: 'center', padding: 16 }}>
              <Spin />
            </div>
          ) : existing.length === 0 ? (
            <div style={{ color: '#999', fontSize: 13, marginTop: 4 }}>
              {t('diagnosis.noPhotosYet')}
            </div>
          ) : (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginTop: 8 }}>
              {existing.map((p) => (
                <div key={p.id} style={{ position: 'relative' }}>
                  <Image
                    src={p.photoUrl}
                    width={88}
                    height={88}
                    style={{ objectFit: 'cover', borderRadius: 6 }}
                  />
                  {!readOnly && (
                    <Popconfirm
                      title={t('diagnosis.deletePhotoConfirm')}
                      okText={t('common.confirm')}
                      cancelText={t('common.cancel')}
                      onConfirm={() => handleDelete(p.id)}
                    >
                      <Button
                        size="small"
                        danger
                        type="primary"
                        shape="circle"
                        icon={<DeleteOutlined />}
                        aria-label={t('diagnosis.deletePhoto')}
                        style={{ position: 'absolute', top: -8, right: -8 }}
                      />
                    </Popconfirm>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Add more photos (respecting the remaining slots) — hidden in read-only mode */}
        {!readOnly && (
        <div>
          <Typography.Text strong>{t('diagnosis.addPhotos')}</Typography.Text>
          <div style={{ color: '#999', fontSize: 12, margin: '2px 0 6px' }}>
            {t('diagnosis.remainingSlots', { count: Math.max(remaining, 0) })}
          </div>
          <input
            type="file"
            accept="image/*"
            multiple
            disabled={remaining <= 0}
            onChange={handleSelect}
          />
          {files.length > 0 && (
            <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>
              {files.map((f) => f.name).join(', ')}
            </div>
          )}
          <Button
            type="primary"
            block
            style={{ marginTop: 8 }}
            disabled={files.length === 0 || files.length > remaining}
            loading={submitting}
            onClick={handleUpload}
          >
            {submitting ? t('diagnosis.uploading') : t('diagnosis.submit')}
          </Button>
        </div>
        )}
      </Space>
    </Modal>
  );
}
