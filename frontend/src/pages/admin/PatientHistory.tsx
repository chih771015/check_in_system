import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Image,
  Skeleton,
  Space,
  Tag,
  Timeline,
  Typography,
  App,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { PatientHistoryEntry, PatientHistoryResponse } from '../../types';
import { getPatientHistory } from '../../api/patients';

const STATUS_COLOR: Record<string, string> = {
  completed: 'green',
  no_show: 'red',
  pending: 'orange',
};

export default function PatientHistory() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [resp, setResp] = useState<PatientHistoryResponse | null>(null);

  useEffect(() => {
    if (!id) return;
    // Reset the spinner when the :id param changes so a re-fetch shows loading.
    // eslint-disable-next-line react-hooks/set-state-in-effect -- intentional re-fetch reset
    setLoading(true);
    getPatientHistory(Number(id))
      .then((r) => setResp(r))
      .catch(() => {
        void message.error(t('errors.INTERNAL_ERROR'));
      })
      .finally(() => setLoading(false));
  }, [id, message, t]);

  function renderEntry(entry: PatientHistoryEntry) {
    return (
      <div>
        <Space wrap>
          <Typography.Text strong>{entry.date}</Typography.Text>
          <Typography.Text type="secondary">
            {entry.startTime} ~ {entry.endTime}
          </Typography.Text>
          <Tag color={STATUS_COLOR[entry.status] ?? 'default'}>{entry.status}</Tag>
        </Space>
        <div style={{ marginTop: 4 }}>
          <Typography.Text>{t('schedules.location')}: {entry.location}</Typography.Text>
          <br />
          <Typography.Text>{t('schedules.translator')}: {entry.translatorName}</Typography.Text>
          {entry.status === 'no_show' && entry.noShowReason && (
            <>
              <br />
              <Typography.Text type="danger">{entry.noShowReason}</Typography.Text>
            </>
          )}
          {entry.diagnosisPhotos && entry.diagnosisPhotos.length > 0 && (
            <div style={{ marginTop: 8 }}>
              {/* Image.PreviewGroup gives click-to-enlarge + arrow navigation. */}
              <Image.PreviewGroup>
                <Space wrap>
                  {entry.diagnosisPhotos.map((url) => (
                    <Image
                      key={url}
                      src={url}
                      width={96}
                      height={96}
                      style={{ objectFit: 'cover', borderRadius: 4, border: '1px solid #eee' }}
                    />
                  ))}
                </Space>
              </Image.PreviewGroup>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/admin/patients')}>
          {t('common.back')}
        </Button>
        <Typography.Title level={4} style={{ margin: 0 }}>{t('patients.history')}</Typography.Title>
      </div>

      {loading ? (
        <Skeleton active />
      ) : !resp ? (
        <Empty description={t('errors.PATIENT_NOT_FOUND')} />
      ) : (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card title={t('patients.title')}>
            <Descriptions column={2} size="small">
              <Descriptions.Item label={t('common.name')}>{resp.patient.name}</Descriptions.Item>
              <Descriptions.Item label={t('common.phone')}>{resp.patient.phone}</Descriptions.Item>
              <Descriptions.Item label={t('patients.idType')}>
                {t(`patients.idTypes.${resp.patient.idType}`)}
              </Descriptions.Item>
              <Descriptions.Item label={t('patients.idNumber')}>{resp.patient.idNumber}</Descriptions.Item>
              <Descriptions.Item label={t('common.createdAt')}>
                {new Date(resp.patient.createdAt).toLocaleString()}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title={t('patients.history')}>
            {resp.history.length === 0 ? (
              <Empty description={t('patients.historyEmpty')} />
            ) : (
              <Timeline
                items={resp.history.map((entry) => ({
                  key: entry.scheduleId,
                  color: STATUS_COLOR[entry.status] ?? 'blue',
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
