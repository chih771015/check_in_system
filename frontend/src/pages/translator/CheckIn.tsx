import { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Alert, Button, Card, Typography, Spin, App } from 'antd';
import {
  CameraOutlined,
  EnvironmentOutlined,
  LoadingOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ScheduleItem } from '../../types';
import { getMySchedules } from '../../api/schedules';
import { checkin } from '../../api/checkins';
import { useGeolocation } from '../../hooks/useGeolocation';

export default function CheckInPage() {
  const { scheduleId, type } = useParams<{ scheduleId: string; type: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const [schedule, setSchedule] = useState<ScheduleItem | null>(null);
  const [selfie, setSelfie] = useState<File | null>(null);
  const [selfiePreview, setSelfiePreview] = useState<string>('');
  const [environment, setEnvironment] = useState<File | null>(null);
  const [environmentPreview, setEnvironmentPreview] = useState<string>('');
  const [submitting, setSubmitting] = useState(false);

  const selfieRef = useRef<HTMLInputElement>(null);
  const envRef = useRef<HTMLInputElement>(null);

  const { state: geoState, latitude, longitude, address, request: requestGeo } = useGeolocation();

  useEffect(() => {
    void (async () => {
      try {
        const list = await getMySchedules();
        const found = list.find((s) => s.id === Number(scheduleId));
        if (found) setSchedule(found);
      } catch {
        message.error(t('errors.INTERNAL_ERROR'));
      }
    })();
  }, [scheduleId, message, t]);

  const handleFileChange = (
    e: React.ChangeEvent<HTMLInputElement>,
    setter: (f: File | null) => void,
    previewSetter: (s: string) => void,
  ) => {
    const file = e.target.files?.[0] ?? null;
    setter(file);
    if (file) previewSetter(URL.createObjectURL(file));
  };

  const handleSubmit = async () => {
    if (!selfie) { message.warning(t('errors.SELFIE_REQUIRED')); return; }
    if (!environment) { message.warning(t('errors.ENVIRONMENT_PHOTO_REQUIRED')); return; }
    if (latitude === null || longitude === null) { message.warning(t('checkin.geo.requesting')); return; }

    setSubmitting(true);
    try {
      const fd = new FormData();
      fd.append('selfie', selfie);
      fd.append('environment', environment);
      fd.append('scheduleId', scheduleId!);
      fd.append('type', type!);
      fd.append('latitude', String(latitude));
      fd.append('longitude', String(longitude));
      fd.append('address', address);
      await checkin(fd);
      message.success(t('common.success'));
      navigate('/my-schedules');
    } catch {
      message.error(t('common.failed'));
    } finally {
      setSubmitting(false);
    }
  };

  if (!schedule) {
    return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
  }

  const typeLabel = type === 'arrive' ? t('checkin.checkinType.arrive') : t('checkin.checkinType.leave');

  return (
    <div style={{ maxWidth: 600, margin: '0 auto' }}>
      <Typography.Title level={4}>{typeLabel}</Typography.Title>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong>{t('nav.mySchedules')}</Typography.Text>
        <div style={{ marginTop: 8, color: '#666' }}>
          <div>{t('schedules.date')}: {schedule.date}</div>
          <div>{t('schedules.startTime')}: {schedule.startTime} - {schedule.endTime}</div>
          <div>{t('schedules.location')}: {schedule.location}</div>
          <div>{t('schedules.patientName')}: {schedule.patientName}</div>
        </div>
      </Card>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong><CameraOutlined /> {t('checkin.takingSelfie')}</Typography.Text>
        <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <input ref={selfieRef} type="file" accept="image/*" capture="user" style={{ display: 'none' }}
              onChange={(e) => handleFileChange(e, setSelfie, setSelfiePreview)} />
            <Button icon={<CameraOutlined />} onClick={() => selfieRef.current?.click()} block>{t('checkins.selfie')}</Button>
            {selfiePreview && (
              <img src={selfiePreview} alt=""
                style={{ width: '100%', maxHeight: 200, objectFit: 'cover', marginTop: 8, borderRadius: 8 }} />
            )}
          </div>
          <div>
            <input ref={envRef} type="file" accept="image/*" capture="environment" style={{ display: 'none' }}
              onChange={(e) => handleFileChange(e, setEnvironment, setEnvironmentPreview)} />
            <Button icon={<CameraOutlined />} onClick={() => envRef.current?.click()} block>{t('checkins.environment')}</Button>
            {environmentPreview && (
              <img src={environmentPreview} alt=""
                style={{ width: '100%', maxHeight: 200, objectFit: 'cover', marginTop: 8, borderRadius: 8 }} />
            )}
          </div>
        </div>
      </Card>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong><EnvironmentOutlined /> GPS</Typography.Text>
        <div style={{ marginTop: 12 }}>
          <GeoStatusBlock state={geoState} address={address} onRequest={requestGeo} />
        </div>
      </Card>

      <Button type="primary" size="large" block loading={submitting} onClick={handleSubmit}
        disabled={geoState !== 'success'}>
        {t('checkin.submit')}
      </Button>
    </div>
  );
}

interface GeoStatusBlockProps {
  state: string;
  address: string;
  onRequest: () => void;
}

export function GeoStatusBlock({ state, address, onRequest }: GeoStatusBlockProps) {
  const { t } = useTranslation();
  switch (state) {
    case 'idle':
      return (
        <Alert
          type="warning"
          showIcon
          icon={<EnvironmentOutlined />}
          message={t('checkin.geo.idle')}
          action={
            <Button type="primary" size="small" onClick={onRequest} style={{ marginTop: 8 }}>
              {t('checkin.requestPermission')}
            </Button>
          }
        />
      );

    case 'requesting':
      return (
        <div style={{ padding: '12px 0', color: '#1677ff', display: 'flex', alignItems: 'center', gap: 8 }}>
          <LoadingOutlined />
          <span>{t('checkin.geo.requesting')}</span>
        </div>
      );

    case 'success':
      return (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#52c41a' }}>
          <CheckCircleOutlined style={{ fontSize: 18 }} />
          <span>{address || t('checkin.geo.success')}</span>
        </div>
      );

    case 'denied':
      return (
        <Alert
          type="error"
          showIcon
          icon={<ExclamationCircleOutlined />}
          message={t('checkin.geo.denied')}
          description={
            <div>
              <ul style={{ margin: '4px 0', paddingLeft: 20 }}>
                <li>{t('checkin.iosInstructions')}</li>
                <li>{t('checkin.androidInstructions')}</li>
                <li>{t('checkin.chromeInstructions')}</li>
              </ul>
            </div>
          }
        />
      );

    case 'timeout':
      return (
        <Alert
          type="warning"
          showIcon
          message={t('checkin.geo.timeout')}
          action={
            <Button size="small" onClick={onRequest} style={{ marginTop: 8 }}>{t('checkin.tryAgain')}</Button>
          }
        />
      );

    case 'unavailable':
    default:
      return (
        <Alert
          type="warning"
          showIcon
          message={t('checkin.geo.unavailable')}
          action={
            <Button size="small" onClick={onRequest} style={{ marginTop: 8 }}>{t('checkin.tryAgain')}</Button>
          }
        />
      );
  }
}
