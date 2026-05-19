import { useCallback, useEffect, useState } from 'react';
import { Table, Tag, Typography, DatePicker, Select, Space, App } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { Dayjs } from 'dayjs';
import { useTranslation } from 'react-i18next';
import { getAuditLogs, type AuditLog } from '../../api/audit';

const { RangePicker } = DatePicker;

const actionColors: Record<string, string> = {
  create_translator: 'green',
  update_translator: 'blue',
  disable_translator: 'red',
  reset_password: 'orange',
  create_schedule: 'green',
  update_schedule: 'blue',
  delete_schedule: 'red',
  delete_schedule_group: 'red',
  update_checkin: 'blue',
  delete_checkin: 'red',
  import_schedules: 'purple',
  create_admin: 'green',
  delete_admin: 'red',
  create_patient: 'green',
  update_patient: 'blue',
  delete_patient: 'red',
};

export default function AuditLogs() {
  const [data, setData] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [actionFilter, setActionFilter] = useState<string | undefined>();
  const [dateRange, setDateRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  const { message } = App.useApp();
  const { t } = useTranslation();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string | number> = { page, pageSize };
      if (actionFilter) params.action = actionFilter;
      if (dateRange?.[0]) params.startDate = dateRange[0].format('YYYY-MM-DD');
      if (dateRange?.[1]) params.endDate = dateRange[1].format('YYYY-MM-DD 23:59:59');
      const resp = await getAuditLogs(params);
      setData(resp.data || []);
      setTotal(resp.total || 0);
    } catch {
      message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, actionFilter, dateRange, message, t]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const allActions = Object.keys(actionColors);

  const columns: ColumnsType<AuditLog> = [
    {
      title: t('audit.action'),
      dataIndex: 'created_at',
      width: 170,
      render: (v) => dayjs(v).format('YYYY-MM-DD HH:mm:ss'),
    },
    { title: t('audit.operator'), dataIndex: 'admin_name', width: 140 },
    {
      title: t('audit.action'),
      dataIndex: 'action',
      width: 200,
      render: (v: string) => <Tag color={actionColors[v] || 'default'}>{v}</Tag>,
    },
    { title: t('audit.targetType'), dataIndex: 'target_type', width: 120 },
    { title: t('common.id'), dataIndex: 'target_id', width: 100 },
    { title: t('audit.detailField'), dataIndex: 'detail', ellipsis: true },
  ];

  return (
    <div>
      <Typography.Title level={4}>{t('audit.title')}</Typography.Title>
      <Space style={{ marginBottom: 16 }}>
        <Select
          allowClear
          placeholder={t('audit.filterAction')}
          style={{ width: 200 }}
          value={actionFilter}
          onChange={(v) => {
            setPage(1);
            setActionFilter(v);
          }}
          options={allActions.map((k) => ({ value: k, label: k }))}
        />
        <RangePicker
          onChange={(range) => {
            setPage(1);
            setDateRange(range);
          }}
        />
      </Space>
      <Table<AuditLog>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={data}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />
    </div>
  );
}
