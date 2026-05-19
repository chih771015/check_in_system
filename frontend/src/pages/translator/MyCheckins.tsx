import { useCallback, useEffect, useState } from 'react';
import { Table, Tag, Typography, DatePicker, Row, Col, Card, Statistic, App } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { Dayjs } from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { CheckinItem } from '../../types';
import { getMyCheckins, getMyCheckinStats, type MyCheckinStats } from '../../api/checkins';
import MapLink from '../../components/MapLink';

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
  const { t } = useTranslation();

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
      message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [dateRange, message, t]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const columns: ColumnsType<CheckinItem> = [
    {
      title: t('checkins.checkinTime'),
      dataIndex: 'checkinTime',
      width: 170,
      render: (v) => dayjs(v).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('common.status'),
      dataIndex: 'type',
      width: 100,
      render: (v: string) =>
        v === 'arrive' ? <Tag color="green">{t('checkins.type.arrive')}</Tag> : <Tag color="blue">{t('checkins.type.leave')}</Tag>,
    },
    {
      title: t('checkins.address'),
      key: 'address',
      ellipsis: true,
      render: (_: unknown, r: CheckinItem) => (
        <MapLink latitude={r.latitude} longitude={r.longitude} address={r.address} />
      ),
    },
    {
      title: t('checkins.isMakeup'),
      dataIndex: 'isMakeup',
      width: 100,
      render: (v) => (v ? <Tag color="orange">{t('checkins.isMakeup')}</Tag> : '-'),
    },
    {
      title: t('checkins.makeupReason'),
      dataIndex: 'makeupReason',
      ellipsis: true,
    },
  ];

  return (
    <div>
      <Typography.Title level={4}>{t('checkins.myTitle')}</Typography.Title>

      <div style={{ marginBottom: 16 }}>
        <RangePicker value={dateRange} onChange={(v) => setDateRange(v)} />
      </div>

      {stats && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={4}><Card><Statistic title={t('checkins.stats.total')} value={stats.total} /></Card></Col>
          <Col span={4}><Card><Statistic title={t('checkins.stats.arrive')} value={stats.arriveCount} /></Card></Col>
          <Col span={4}><Card><Statistic title={t('checkins.stats.leave')} value={stats.leaveCount} /></Card></Col>
          <Col span={4}><Card><Statistic title={t('checkins.stats.makeup')} value={stats.makeupCount} valueStyle={{ color: '#fa8c16' }} /></Card></Col>
          <Col span={4}><Card><Statistic title={t('checkins.stats.onTime')} value={stats.onTimeCount} valueStyle={{ color: '#52c41a' }} /></Card></Col>
          <Col span={4}><Card><Statistic title={t('checkins.stats.late')} value={stats.lateCount} valueStyle={{ color: '#f5222d' }} /></Card></Col>
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
