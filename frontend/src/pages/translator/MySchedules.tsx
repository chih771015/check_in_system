import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, Button, Tag, Switch, Typography, Spin, Empty, App } from 'antd';
import {
  EnvironmentOutlined,
  ClockCircleOutlined,
  UserOutlined,
  CheckCircleFilled,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ScheduleItem } from '../../types';
import { getMySchedules } from '../../api/schedules';

const statusColorMap: Record<string, string> = {
  none: 'default',
  arrived: 'orange',
  completed: 'green',
  makeup: 'blue',
};

const statusLabelKey: Record<string, string> = {
  none: 'checkins.type.arrive',
  arrived: 'checkins.type.arrive',
  completed: 'common.success',
  makeup: 'checkins.isMakeup',
};

export default function MySchedules() {
  const [schedules, setSchedules] = useState<ScheduleItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const fetchSchedules = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (!showHistory) params.date_from = new Date().toISOString().slice(0, 10);
      const list = await getMySchedules(params);
      setSchedules(list);
    } catch {
      message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [showHistory, message, t]);

  useEffect(() => {
    void fetchSchedules();
  }, [fetchSchedules]);

  const isPast = (schedule: ScheduleItem) => {
    const now = new Date();
    const today = now.toISOString().slice(0, 10);
    if (schedule.date < today) return true;
    if (schedule.date > today) return false;
    const [endH, endM] = schedule.endTime.split(':').map(Number);
    const endAt = new Date();
    endAt.setHours(endH, endM, 0, 0);
    return now > endAt;
  };

  const renderActions = (schedule: ScheduleItem) => {
    const past = isPast(schedule);

    if (schedule.checkinStatus === 'completed') {
      return <Tag icon={<CheckCircleFilled />} color="green">{t('common.success')}</Tag>;
    }
    if (schedule.checkinStatus === 'none' && !past) {
      return (
        <Button type="primary" onClick={() => navigate(`/checkin/${schedule.id}/arrive`)}>
          {t('checkin.checkinType.arrive')}
        </Button>
      );
    }
    if (schedule.checkinStatus === 'arrived' && !past) {
      return (
        <Button type="primary" onClick={() => navigate(`/checkin/${schedule.id}/leave`)}>
          {t('checkin.checkinType.leave')}
        </Button>
      );
    }
    if (schedule.checkinStatus === 'none' && past) {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/arrive`)}>
          {t('checkins.isMakeup')} ({t('checkins.type.arrive')})
        </Button>
      );
    }
    if (schedule.checkinStatus === 'arrived' && past) {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/leave`)}>
          {t('checkins.isMakeup')} ({t('checkins.type.leave')})
        </Button>
      );
    }
    if (schedule.checkinStatus === 'makeup') {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/leave`)}>
          {t('checkins.isMakeup')} ({t('checkins.type.leave')})
        </Button>
      );
    }
    return null;
  };

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Typography.Title level={4} style={{ margin: 0 }}>
          {t('nav.mySchedules')}
        </Typography.Title>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Typography.Text>{t('schedules.showHistory')}</Typography.Text>
          <Switch checked={showHistory} onChange={setShowHistory} />
        </div>
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : schedules.length === 0 ? (
        <Empty />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {schedules.map((s) => (
            <Card key={s.id} size="small">
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'flex-start',
                  flexWrap: 'wrap',
                  gap: 8,
                }}
              >
                <div style={{ flex: 1, minWidth: 200 }}>
                  <div style={{ marginBottom: 4 }}>
                    <Typography.Text strong>{s.date}</Typography.Text>
                    <Tag color={statusColorMap[s.checkinStatus] ?? 'default'} style={{ marginLeft: 8 }}>
                      {t(statusLabelKey[s.checkinStatus] ?? 'common.status')}
                    </Tag>
                  </div>
                  <div style={{ color: '#666', fontSize: 14 }}>
                    <div>
                      <ClockCircleOutlined style={{ marginRight: 4 }} />
                      {s.startTime} - {s.endTime}
                    </div>
                    <div>
                      <EnvironmentOutlined style={{ marginRight: 4 }} />
                      {s.location}
                    </div>
                    <div>
                      <UserOutlined style={{ marginRight: 4 }} />
                      {s.patientName}
                    </div>
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  {renderActions(s)}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
