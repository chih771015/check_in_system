import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, Button, Tag, Switch, Typography, Spin, Empty, App } from 'antd';
import {
  EnvironmentOutlined,
  ClockCircleOutlined,
  UserOutlined,
  CheckCircleFilled,
} from '@ant-design/icons';
import type { ScheduleItem } from '../../types';
import { getMySchedules } from '../../api/schedules';

const statusConfig: Record<string, { color: string; label: string }> = {
  none: { color: 'default', label: '未打卡' },
  arrived: { color: 'orange', label: '已到達' },
  completed: { color: 'green', label: '已完成' },
  makeup: { color: 'blue', label: '補打卡' },
};

export default function MySchedules() {
  const [schedules, setSchedules] = useState<ScheduleItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const navigate = useNavigate();
  const { message } = App.useApp();

  const fetchSchedules = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (!showHistory) params.upcoming = 'true';
      const list = await getMySchedules(params);
      setSchedules(list);
    } catch {
      message.error('載入排班失敗');
    } finally {
      setLoading(false);
    }
  }, [showHistory, message]);

  useEffect(() => {
    void fetchSchedules();
  }, [fetchSchedules]);

  const isPast = (schedule: ScheduleItem) => {
    const today = new Date().toISOString().slice(0, 10);
    return schedule.date < today;
  };

  const renderActions = (schedule: ScheduleItem) => {
    const past = isPast(schedule);

    if (schedule.checkinStatus === 'completed') {
      return (
        <Tag icon={<CheckCircleFilled />} color="green">
          已完成
        </Tag>
      );
    }

    if (schedule.checkinStatus === 'none' && !past) {
      return (
        <Button
          type="primary"
          onClick={() => navigate(`/checkin/${schedule.id}/arrive`)}
        >
          到達打卡
        </Button>
      );
    }

    if (schedule.checkinStatus === 'arrived' && !past) {
      return (
        <Button
          type="primary"
          onClick={() => navigate(`/checkin/${schedule.id}/leave`)}
        >
          離開打卡
        </Button>
      );
    }

    if (schedule.checkinStatus === 'none' && past) {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/arrive`)}>
          補打卡(到達)
        </Button>
      );
    }

    if (schedule.checkinStatus === 'arrived' && past) {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/leave`)}>
          補打卡(離開)
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
          我的排班
        </Typography.Title>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Typography.Text>顯示歷史</Typography.Text>
          <Switch checked={showHistory} onChange={setShowHistory} />
        </div>
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : schedules.length === 0 ? (
        <Empty description="目前沒有排班" />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {schedules.map((s) => {
            const info = statusConfig[s.checkinStatus] ?? statusConfig.none;
            return (
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
                      <Tag color={info.color} style={{ marginLeft: 8 }}>
                        {info.label}
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
            );
          })}
        </div>
      )}
    </div>
  );
}
