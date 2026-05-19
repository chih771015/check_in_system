import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Skeleton,
  Space,
  Tag,
  Timeline,
  Typography,
  App,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import type { IDType, PatientHistoryEntry, PatientHistoryResponse } from '../../types';
import { getPatientHistory } from '../../api/patients';

const ID_TYPE_LABEL: Record<IDType, string> = {
  passport: '護照',
  hn: '病歷號 (HN)',
  unid: '識別號 (UNID)',
};

const STATUS_LABEL: Record<string, { text: string; color: string }> = {
  completed: { text: '已上傳診斷證明', color: 'green' },
  no_show: { text: '未到', color: 'red' },
  pending: { text: '待處理', color: 'orange' },
};

function renderEntry(entry: PatientHistoryEntry) {
  const status = STATUS_LABEL[entry.status] ?? { text: entry.status, color: 'default' };
  return (
    <div>
      <Space wrap>
        <Typography.Text strong>{entry.date}</Typography.Text>
        <Typography.Text type="secondary">
          {entry.startTime} ~ {entry.endTime}
        </Typography.Text>
        <Tag color={status.color}>{status.text}</Tag>
      </Space>
      <div style={{ marginTop: 4 }}>
        <Typography.Text>地點：{entry.location}</Typography.Text>
        <br />
        <Typography.Text>翻譯員：{entry.translatorName}</Typography.Text>
        {entry.status === 'no_show' && entry.noShowReason && (
          <>
            <br />
            <Typography.Text type="danger">未到原因：{entry.noShowReason}</Typography.Text>
          </>
        )}
        {entry.diagnosisPhotos && entry.diagnosisPhotos.length > 0 && (
          <div style={{ marginTop: 8, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {entry.diagnosisPhotos.map((url) => (
              <img
                key={url}
                src={url}
                alt="診斷證明"
                style={{ width: 96, height: 96, objectFit: 'cover', borderRadius: 4, border: '1px solid #eee' }}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default function PatientHistory() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [resp, setResp] = useState<PatientHistoryResponse | null>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    getPatientHistory(Number(id))
      .then((r) => setResp(r))
      .catch(() => {
        void message.error('載入歷史紀錄失敗');
      })
      .finally(() => setLoading(false));
  }, [id, message]);

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/admin/patients')}>
          返回列表
        </Button>
        <Typography.Title level={4} style={{ margin: 0 }}>病人歷史看診紀錄</Typography.Title>
      </div>

      {loading ? (
        <Skeleton active />
      ) : !resp ? (
        <Empty description="找不到病人資料" />
      ) : (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card title="病人基本資料">
            <Descriptions column={2} size="small">
              <Descriptions.Item label="姓名">{resp.patient.name}</Descriptions.Item>
              <Descriptions.Item label="電話">{resp.patient.phone}</Descriptions.Item>
              <Descriptions.Item label="ID 類型">
                {ID_TYPE_LABEL[resp.patient.idType]}
              </Descriptions.Item>
              <Descriptions.Item label="ID 號碼">{resp.patient.idNumber}</Descriptions.Item>
              <Descriptions.Item label="建立時間">
                {new Date(resp.patient.createdAt).toLocaleString('zh-TW')}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="看診紀錄（依時間倒序）">
            {resp.history.length === 0 ? (
              <Empty description="No visit records yet. This feature will be available in a future release." />
            ) : (
              <Timeline
                items={resp.history.map((entry) => ({
                  key: entry.scheduleId,
                  color: STATUS_LABEL[entry.status]?.color ?? 'blue',
                  children: renderEntry(entry),
                }))}
              />
            )}
          </Card>
        </Space>
      )}
    </div>
  );
}
