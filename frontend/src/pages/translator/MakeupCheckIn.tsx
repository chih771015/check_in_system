import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Button, Card, Typography, Spin, Input, App } from 'antd';
import {
  CameraOutlined,
  EnvironmentOutlined,
  LoadingOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import type { ScheduleItem } from '../../types';
import { getMySchedules } from '../../api/schedules';
import { makeupCheckin } from '../../api/checkins';

export default function MakeupCheckInPage() {
  const { scheduleId, type } = useParams<{ scheduleId: string; type: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();

  const [schedule, setSchedule] = useState<ScheduleItem | null>(null);
  const [selfie, setSelfie] = useState<File | null>(null);
  const [selfiePreview, setSelfiePreview] = useState<string>('');
  const [environment, setEnvironment] = useState<File | null>(null);
  const [environmentPreview, setEnvironmentPreview] = useState<string>('');
  const [latitude, setLatitude] = useState<number | null>(null);
  const [longitude, setLongitude] = useState<number | null>(null);
  const [address, setAddress] = useState<string>('');
  const [gpsLoading, setGpsLoading] = useState(true);
  const [gpsError, setGpsError] = useState<string>('');
  const [submitting, setSubmitting] = useState(false);
  const [makeupReason, setMakeupReason] = useState<string>('');

  const selfieRef = useRef<HTMLInputElement>(null);
  const envRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    void (async () => {
      try {
        const list = await getMySchedules();
        const found = list.find((s) => s.id === Number(scheduleId));
        if (found) setSchedule(found);
      } catch {
        message.error('載入排班資訊失敗');
      }
    })();
  }, [scheduleId, message]);

  const fetchGps = useCallback(() => {
    setGpsLoading(true);
    setGpsError('');
    if (!navigator.geolocation) {
      setGpsError('瀏覽器不支援定位功能');
      setGpsLoading(false);
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setLatitude(pos.coords.latitude);
        setLongitude(pos.coords.longitude);
        setAddress(`${pos.coords.latitude.toFixed(6)}, ${pos.coords.longitude.toFixed(6)}`);
        setGpsLoading(false);
      },
      (err) => {
        setGpsError(`定位失敗: ${err.message}`);
        setGpsLoading(false);
      },
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }, []);

  useEffect(() => {
    fetchGps();
  }, [fetchGps]);

  const handleFileChange = (
    e: React.ChangeEvent<HTMLInputElement>,
    setter: (f: File | null) => void,
    previewSetter: (s: string) => void,
  ) => {
    const file = e.target.files?.[0] ?? null;
    setter(file);
    if (file) {
      const url = URL.createObjectURL(file);
      previewSetter(url);
    }
  };

  const handleSubmit = async () => {
    if (!selfie) {
      message.warning('請拍攝自拍照');
      return;
    }
    if (!environment) {
      message.warning('請拍攝環境照');
      return;
    }
    if (latitude === null || longitude === null) {
      message.warning('尚未取得定位資訊');
      return;
    }
    if (!makeupReason.trim()) {
      message.warning('請填寫補打卡原因');
      return;
    }

    setSubmitting(true);
    try {
      const fd = new FormData();
      fd.append('selfie', selfie);
      fd.append('environment', environment);
      fd.append('schedule_id', scheduleId!);
      fd.append('type', type!);
      fd.append('latitude', String(latitude));
      fd.append('longitude', String(longitude));
      fd.append('address', address);
      fd.append('makeup_reason', makeupReason);
      await makeupCheckin(fd);
      message.success('補打卡成功');
      navigate('/my-schedules');
    } catch {
      message.error('補打卡失敗');
    } finally {
      setSubmitting(false);
    }
  };

  const typeLabel = type === 'arrive' ? '補打卡(到達)' : '補打卡(離開)';

  if (!schedule) {
    return (
      <div style={{ textAlign: 'center', padding: 48 }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 600, margin: '0 auto' }}>
      <Typography.Title level={4}>{typeLabel}</Typography.Title>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong>排班資訊</Typography.Text>
        <div style={{ marginTop: 8, color: '#666' }}>
          <div>日期: {schedule.date}</div>
          <div>
            時間: {schedule.startTime} - {schedule.endTime}
          </div>
          <div>地點: {schedule.location}</div>
          <div>病患: {schedule.patientName}</div>
        </div>
      </Card>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong>補打卡原因</Typography.Text>
        <Input.TextArea
          rows={3}
          placeholder="請輸入補打卡原因（必填）"
          value={makeupReason}
          onChange={(e) => setMakeupReason(e.target.value)}
          style={{ marginTop: 8 }}
        />
      </Card>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong>
          <CameraOutlined /> 拍照
        </Typography.Text>
        <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <input
              ref={selfieRef}
              type="file"
              accept="image/*"
              capture="user"
              style={{ display: 'none' }}
              onChange={(e) => handleFileChange(e, setSelfie, setSelfiePreview)}
            />
            <Button
              icon={<CameraOutlined />}
              onClick={() => selfieRef.current?.click()}
              block
            >
              拍攝自拍
            </Button>
            {selfiePreview && (
              <img
                src={selfiePreview}
                alt="自拍預覽"
                style={{ width: '100%', maxHeight: 200, objectFit: 'cover', marginTop: 8, borderRadius: 8 }}
              />
            )}
          </div>
          <div>
            <input
              ref={envRef}
              type="file"
              accept="image/*"
              capture="environment"
              style={{ display: 'none' }}
              onChange={(e) => handleFileChange(e, setEnvironment, setEnvironmentPreview)}
            />
            <Button
              icon={<CameraOutlined />}
              onClick={() => envRef.current?.click()}
              block
            >
              拍攝環境照
            </Button>
            {environmentPreview && (
              <img
                src={environmentPreview}
                alt="環境照預覽"
                style={{ width: '100%', maxHeight: 200, objectFit: 'cover', marginTop: 8, borderRadius: 8 }}
              />
            )}
          </div>
        </div>
      </Card>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong>
          <EnvironmentOutlined /> 定位資訊
        </Typography.Text>
        <div style={{ marginTop: 8 }}>
          {gpsLoading ? (
            <div>
              <LoadingOutlined /> 正在取得定位...
            </div>
          ) : gpsError ? (
            <div>
              <Typography.Text type="danger">{gpsError}</Typography.Text>
              <Button size="small" style={{ marginLeft: 8 }} onClick={fetchGps}>
                重試
              </Button>
            </div>
          ) : (
            <div>
              <CheckCircleOutlined style={{ color: 'green', marginRight: 4 }} />
              {address}
            </div>
          )}
        </div>
      </Card>

      <Button type="primary" size="large" block loading={submitting} onClick={handleSubmit}>
        確認送出
      </Button>
    </div>
  );
}
