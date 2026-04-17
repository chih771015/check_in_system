import { useCallback, useEffect, useState } from 'react';
import { Table, Tag, Typography, DatePicker, Row, Col, Card, Statistic, App } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { Dayjs } from 'dayjs';
import type { CheckinItem } from '../../types';
import { getMyCheckins, getMyCheckinStats, type MyCheckinStats } from '../../api/checkins';

const { RangePicker } = DatePicker;

export default function MyCheckinsPage() {
  const [data, setData] = useState<CheckinItem[]>([]);
  const [stats, setStats] = useState<MyCheckinStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null] | null>([
    dayjs().startOf('month'),
    dayjs().endOf('month'),
  ]);
  const { message } = App.useApp();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (dateRange?.[0]) params.dateFrom = dateRange[0].format('YYYY-MM-DD');
      if (dateRange?.[1]) params.dateTo = dateRange[1].format('YYYY-MM-DD');
      const [list, s] = await Promise.all([getMyCheckins(params), getMyCheckinStats(params)]);
      setData(list || []);
      setStats(s);
    } catch {
      message.error('載入打卡紀錄失敗');
    } finally {
      setLoading(false);
    }
  }, [dateRange, message]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const columns: ColumnsType<CheckinItem> = [
    {
      title: '時間',
      dataIndex: 'checkinTime',
      width: 170,
      render: (v) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '類型',
      dataIndex: 'type',
      width: 100,
      render: (v: string) =>
        v === 'arrive' ? <Tag color="green">到達</Tag> : <Tag color="blue">離開</Tag>,
    },
    {
      title: '地址',
      dataIndex: 'address',
      ellipsis: true,
    },
    {
      title: '補打卡',
      dataIndex: 'isMakeup',
      width: 100,
      render: (v) => (v ? <Tag color="orange">補打卡</Tag> : '-'),
    },
    {
      title: '備註',
      dataIndex: 'makeupReason',
      ellipsis: true,
    },
  ];

  return (
    <div>
      <Typography.Title level={4}>我的打卡紀錄</Typography.Title>

      <div style={{ marginBottom: 16 }}>
        <RangePicker value={dateRange} onChange={(v) => setDateRange(v)} />
      </div>

      {stats && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={4}>
            <Card>
              <Statistic title="總打卡數" value={stats.total} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic title="到達" value={stats.arriveCount} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic title="離開" value={stats.leaveCount} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic title="補打卡" value={stats.makeupCount} valueStyle={{ color: '#fa8c16' }} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic title="準時" value={stats.onTimeCount} valueStyle={{ color: '#52c41a' }} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic title="遲到" value={stats.lateCount} valueStyle={{ color: '#f5222d' }} />
            </Card>
          </Col>
        </Row>
      )}

      <Table<CheckinItem>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={data}
      />
    </div>
  );
}
