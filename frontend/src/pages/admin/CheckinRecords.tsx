import { useCallback, useEffect, useState } from 'react';
import {
  Table,
  Button,
  Select,
  DatePicker,
  Tag,
  Space,
  Modal,
  Image,
  Typography,
  Form,
  Input,
  App,
} from 'antd';
import MapLink from '../../components/MapLink';
import { DownloadOutlined, FileTextOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { CheckinItem, TranslatorListItem } from '../../types';
import {
  getAdminCheckins,
  exportCheckinExcel,
  exportCheckinGoogleSheet,
  updateCheckin,
  deleteCheckin,
} from '../../api/checkins';
import { getTranslators } from '../../api/translators';

const { RangePicker } = DatePicker;

const typeColorMap: Record<string, string> = {
  arrive: 'blue',
  leave: 'green',
};

export default function CheckinRecords() {
  const [data, setData] = useState<CheckinItem[]>([]);
  const [translators, setTranslators] = useState<TranslatorListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportingSheet, setExportingSheet] = useState(false);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [detailRecord, setDetailRecord] = useState<CheckinItem | null>(null);
  const [editRecord, setEditRecord] = useState<CheckinItem | null>(null);
  const [editForm] = Form.useForm();
  const { message, modal } = App.useApp();
  const { t } = useTranslation();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getAdminCheckins(filters);
      setData(Array.isArray(list) ? list : []);
    } catch {
      message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [filters, message, t]);

  useEffect(() => {
    void fetchData();
    void getTranslators().then(setTranslators).catch(() => undefined);
  }, [fetchData]);

  const handleExport = async () => {
    setExporting(true);
    try {
      await exportCheckinExcel(filters);
      message.success(t('common.success'));
    } catch {
      message.error(t('errors.EXPORT_FAILED'));
    } finally {
      setExporting(false);
    }
  };

  const openEditCheckin = (record: CheckinItem) => {
    setEditRecord(record);
    editForm.setFieldsValue({
      checkinTime: record.checkinTime ? record.checkinTime.slice(0, 16) : '',
      address: record.address,
      makeupReason: record.makeupReason,
    });
  };

  const handleEditCheckin = async (values: {
    checkinTime: string;
    address: string;
    makeupReason: string;
  }) => {
    if (!editRecord) return;
    try {
      await updateCheckin(editRecord.id, {
        checkinTime: values.checkinTime ? new Date(values.checkinTime).toISOString() : undefined,
        address: values.address,
        makeupReason: values.makeupReason,
      });
      message.success(t('common.success'));
      setEditRecord(null);
      void fetchData();
    } catch {
      message.error(t('common.failed'));
    }
  };

  const handleDeleteCheckin = (record: CheckinItem) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('schedules.confirmDelete'),
      okText: t('common.delete'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deleteCheckin(record.id);
          message.success(t('common.success'));
          void fetchData();
        } catch {
          message.error(t('common.failed'));
        }
      },
    });
  };

  const handleGoogleSheet = async () => {
    setExportingSheet(true);
    try {
      const res = await exportCheckinGoogleSheet(filters);
      message.success(t('common.success'));
      window.open(res.url, '_blank');
    } catch (err: unknown) {
      const code = (err as { response?: { data?: { code?: string } } })?.response?.data?.code;
      if (code === 'GOOGLE_NOT_CONFIGURED') {
        message.warning(t('errors.GOOGLE_NOT_CONFIGURED'));
      } else {
        message.error(t('errors.EXPORT_FAILED'));
      }
    } finally {
      setExportingSheet(false);
    }
  };

  const handleDateChange = (_: unknown, dateStrings: [string, string]) => {
    if (dateStrings[0] && dateStrings[1]) {
      setFilters((prev) => ({ ...prev, dateFrom: dateStrings[0], dateTo: dateStrings[1] }));
    } else {
      setFilters((prev) => {
        const next = { ...prev };
        delete next.dateFrom;
        delete next.dateTo;
        return next;
      });
    }
  };

  const columns = [
    {
      title: t('checkins.checkinTime'),
      dataIndex: 'checkinTime',
      key: 'checkinTime',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    { title: t('schedules.translator'), dataIndex: 'translatorName', key: 'translatorName', width: 100 },
    {
      title: t('common.status'),
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (v: string) => (
        <Tag color={typeColorMap[v] ?? 'default'}>{t(`checkins.type.${v}`)}</Tag>
      ),
    },
    {
      title: t('checkins.address'),
      key: 'address',
      render: (_: unknown, r: CheckinItem) => (
        <MapLink latitude={r.latitude} longitude={r.longitude} address={r.address} />
      ),
    },
    {
      title: t('checkins.isMakeup'),
      dataIndex: 'isMakeup',
      key: 'isMakeup',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="orange">{t('common.yes')}</Tag> : <Tag>{t('common.no')}</Tag>),
    },
    {
      title: t('common.operation'),
      key: 'action',
      width: 180,
      render: (_: unknown, record: CheckinItem) => (
        <Space>
          <Button size="small" onClick={() => setDetailRecord(record)}>{t('common.detail')}</Button>
          <Button size="small" onClick={() => openEditCheckin(record)}>{t('common.edit')}</Button>
          <Button size="small" danger onClick={() => handleDeleteCheckin(record)}>{t('common.delete')}</Button>
        </Space>
      ),
    },
  ];

  const translatorOptions = translators.map((tr) => ({ value: String(tr.id), label: tr.name }));

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
        <RangePicker onChange={handleDateChange} />
        <Select
          style={{ width: 140 }}
          allowClear
          placeholder={t('schedules.translator')}
          options={translatorOptions}
          onChange={(v) =>
            setFilters((prev) => {
              const next = { ...prev };
              if (v) next.translatorId = v;
              else delete next.translatorId;
              return next;
            })
          }
        />
        <Select
          style={{ width: 120 }}
          allowClear
          placeholder={t('checkins.filterType')}
          options={[
            { value: 'arrive', label: t('checkins.type.arrive') },
            { value: 'leave', label: t('checkins.type.leave') },
          ]}
          onChange={(v) =>
            setFilters((prev) => {
              const next = { ...prev };
              if (v) next.type = v;
              else delete next.type;
              return next;
            })
          }
        />
        <Select
          style={{ width: 130 }}
          allowClear
          placeholder={t('checkins.isMakeup')}
          options={[
            { value: 'true', label: t('common.yes') },
            { value: 'false', label: t('common.no') },
          ]}
          onChange={(v) =>
            setFilters((prev) => {
              const next = { ...prev };
              if (v !== undefined) next.isMakeup = v;
              else delete next.isMakeup;
              return next;
            })
          }
        />
        <div style={{ flex: 1 }} />
        <Button icon={<FileTextOutlined />} loading={exportingSheet} onClick={handleGoogleSheet}>
          {t('checkins.exportGoogleSheet')}
        </Button>
        <Button icon={<DownloadOutlined />} loading={exporting} onClick={handleExport}>
          {t('checkins.exportExcel')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        scroll={{ x: 700 }}
        pagination={{ pageSize: 20 }}
      />

      <Modal
        title={t('checkins.detailModal')}
        open={!!detailRecord}
        onCancel={() => setDetailRecord(null)}
        footer={null}
        width={600}
      >
        {detailRecord && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Typography.Text type="secondary">{t('schedules.translator')}</Typography.Text>
              <div>{detailRecord.translatorName}（ID: {detailRecord.translatorId}）</div>
            </div>
            <div>
              <Typography.Text type="secondary">{t('common.status')}</Typography.Text>
              <div>
                <Tag color={typeColorMap[detailRecord.type]}>{t(`checkins.type.${detailRecord.type}`)}</Tag>
                {detailRecord.isMakeup && <Tag color="orange">{t('checkins.isMakeup')}</Tag>}
              </div>
            </div>
            <div>
              <Typography.Text type="secondary">{t('checkins.checkinTime')}</Typography.Text>
              <div>{new Date(detailRecord.checkinTime).toLocaleString()}</div>
            </div>
            <div>
              <Typography.Text type="secondary">GPS</Typography.Text>
              <div>
                <MapLink
                  latitude={detailRecord.latitude}
                  longitude={detailRecord.longitude}
                  address={detailRecord.address}
                />
              </div>
              <div style={{ color: '#999', fontSize: 12, marginTop: 2 }}>
                {detailRecord.latitude.toFixed(6)}, {detailRecord.longitude.toFixed(6)}
              </div>
            </div>
            {detailRecord.isMakeup && (
              <div>
                <Typography.Text type="secondary">{t('checkins.makeupReason')}</Typography.Text>
                <div>{detailRecord.makeupReason}</div>
              </div>
            )}
            <div>
              <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                <div>
                  <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>{t('checkins.selfie')}</div>
                  <Image
                    src={`http://localhost:8080${detailRecord.selfieUrl}`}
                    width={200}
                    style={{ borderRadius: 8 }}
                    fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
                  />
                </div>
                {detailRecord.environmentUrl && (
                  <div>
                    <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>{t('checkins.environment')}</div>
                    <Image
                      src={`http://localhost:8080${detailRecord.environmentUrl}`}
                      width={200}
                      style={{ borderRadius: 8 }}
                      fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
                    />
                  </div>
                )}
              </div>
            </div>
          </Space>
        )}
      </Modal>

      <Modal
        title={t('common.edit')}
        open={!!editRecord}
        onCancel={() => setEditRecord(null)}
        footer={null}
      >
        {editRecord && (
          <Form form={editForm} onFinish={handleEditCheckin} layout="vertical">
            <Form.Item name="checkinTime" label={t('checkins.checkinTime')}>
              <Input type="datetime-local" />
            </Form.Item>
            <Form.Item name="address" label={t('checkins.address')}>
              <Input />
            </Form.Item>
            {editRecord.isMakeup && (
              <Form.Item name="makeupReason" label={t('checkins.makeupReason')}>
                <Input.TextArea rows={2} />
              </Form.Item>
            )}
            <Form.Item>
              <Button type="primary" htmlType="submit" block>{t('common.update')}</Button>
            </Form.Item>
          </Form>
        )}
      </Modal>
    </div>
  );
}
