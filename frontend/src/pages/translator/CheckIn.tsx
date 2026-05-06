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
import type { ScheduleItem } from '../../types';
import { getMySchedules } from '../../api/schedules';
import { checkin } from '../../api/checkins';
import { useGeolocation } from '../../hooks/useGeolocation';

export default function CheckInPage() {
  const { scheduleId, type } = useParams<{ scheduleId: string; type: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();

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
        message.error('載入排班資訊失敗');
      }
    })();
  }, [scheduleId, message]);

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
    if (!selfie) { message.warning('請拍攝自拍照'); return; }
    if (!environment) { message.warning('請拍攝環境照'); return; }
    if (latitude === null || longitude === null) { message.warning('尚未取得定位資訊'); return; }

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
      message.success('打卡成功');
      navigate('/my-schedules');
    } catch {
      message.error('打卡失敗');
    } finally {
      setSubmitting(false);
    }
  };

  if (!schedule) {
    return <div style={{ textAlign: 'center', padding: 48 }}><Spin size="large" /></div>;
  }

  const typeLabel = type === 'arrive' ? '到達打卡' : '離開打卡';

  return (
    <div style={{ maxWidth: 600, margin: '0 auto' }}>
      <Typography.Title level={4}>{typeLabel}</Typography.Title>

      {/* 排班資訊 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong>排班資訊</Typography.Text>
        <div style={{ marginTop: 8, color: '#666' }}>
          <div>日期: {schedule.date}</div>
          <div>時間: {schedule.startTime} - {schedule.endTime}</div>
          <div>地點: {schedule.location}</div>
          <div>病患: {schedule.patientName}</div>
        </div>
      </Card>

      {/* 拍照 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong><CameraOutlined /> 拍照</Typography.Text>
        <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div>
            <input ref={selfieRef} type="file" accept="image/*" capture="user" style={{ display: 'none' }}
              onChange={(e) => handleFileChange(e, setSelfie, setSelfiePreview)} />
            <Button icon={<CameraOutlined />} onClick={() => selfieRef.current?.click()} block>拍攝自拍</Button>
            {selfiePreview && (
              <img src={selfiePreview} alt="自拍預覽"
                style={{ width: '100%', maxHeight: 200, objectFit: 'cover', marginTop: 8, borderRadius: 8 }} />
            )}
          </div>
          <div>
            <input ref={envRef} type="file" accept="image/*" capture="environment" style={{ display: 'none' }}
              onChange={(e) => handleFileChange(e, setEnvironment, setEnvironmentPreview)} />
            <Button icon={<CameraOutlined />} onClick={() => envRef.current?.click()} block>拍攝環境照</Button>
            {environmentPreview && (
              <img src={environmentPreview} alt="環境照預覽"
                style={{ width: '100%', maxHeight: 200, objectFit: 'cover', marginTop: 8, borderRadius: 8 }} />
            )}
          </div>
        </div>
      </Card>

      {/* 定位 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Typography.Text strong><EnvironmentOutlined /> 定位資訊</Typography.Text>
        <div style={{ marginTop: 12 }}>
          <GeoStatusBlock state={geoState} address={address} onRequest={requestGeo} />
        </div>
      </Card>

      <Button type="primary" size="large" block loading={submitting} onClick={handleSubmit}
        disabled={geoState !== 'success'}>
        確認送出
      </Button>
    </div>
  );
}

// ── 共用定位狀態區塊 ────────────────────────────────────────────────────────
interface GeoStatusBlockProps {
  state: string;
  address: string;
  onRequest: () => void;
}

export function GeoStatusBlock({ state, address, onRequest }: GeoStatusBlockProps) {
  switch (state) {
    case 'idle':
      return (
        <Alert
          type="warning"
          showIcon
          icon={<EnvironmentOutlined />}
          message="需要定位權限"
          description="打卡需要記錄您的位置。點下方按鈕後，瀏覽器會詢問是否允許，請選擇「允許」。"
          action={
            <Button type="primary" size="small" onClick={onRequest} style={{ marginTop: 8 }}>
              授權定位
            </Button>
          }
        />
      );

    case 'requesting':
      return (
        <div style={{ padding: '12px 0', color: '#1677ff', display: 'flex', alignItems: 'center', gap: 8 }}>
          <LoadingOutlined />
          <span>正在取得定位，請稍候…</span>
        </div>
      );

    case 'success':
      return (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#52c41a' }}>
          <CheckCircleOutlined style={{ fontSize: 18 }} />
          <span>{address}</span>
        </div>
      );

    case 'denied':
      return (
        <Alert
          type="error"
          showIcon
          icon={<ExclamationCircleOutlined />}
          message="定位權限已被拒絕"
          description={
            <div>
              <p style={{ margin: '4px 0' }}>請按照以下步驟手動開啟：</p>
              <ul style={{ margin: '4px 0', paddingLeft: 20 }}>
                <li><strong>iOS Safari：</strong>設定 → Safari → 位置 → 允許</li>
                <li><strong>Android Chrome：</strong>點網址列左側鎖頭 → 權限 → 位置 → 允許</li>
                <li><strong>電腦 Chrome：</strong>點網址列右側 🔒 → 位置 → 允許</li>
              </ul>
              <p style={{ margin: '4px 0' }}>開啟後請重新整理此頁面。</p>
            </div>
          }
        />
      );

    case 'timeout':
      return (
        <Alert
          type="warning"
          showIcon
          message="定位超時"
          description="取得位置時間過長，可能是 GPS 訊號不佳。請移至戶外或靠近窗戶後重試。"
          action={
            <Button size="small" onClick={onRequest} style={{ marginTop: 8 }}>重試</Button>
          }
        />
      );

    case 'unavailable':
    default:
      return (
        <Alert
          type="warning"
          showIcon
          message="無法取得定位"
          description="裝置目前無法取得位置資訊，請確認 GPS 已開啟或連上網路後重試。"
          action={
            <Button size="small" onClick={onRequest} style={{ marginTop: 8 }}>重試</Button>
          }
        />
      );
  }
}
