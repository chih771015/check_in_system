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
  App,
} from 'antd';
import { DownloadOutlined, FileTextOutlined } from '@ant-design/icons';
import type { CheckinItem, TranslatorListItem } from '../../types';
import { getAdminCheckins, exportCheckinExcel, exportCheckinGoogleSheet } from '../../api/checkins';
import { getTranslators } from '../../api/translators';

const { RangePicker } = DatePicker;

const typeTagMap: Record<string, { color: string; label: string }> = {
  arrive: { color: 'blue', label: '到達' },
  leave: { color: 'green', label: '離開' },
};

export default function CheckinRecords() {
  const [data, setData] = useState<CheckinItem[]>([]);
  const [translators, setTranslators] = useState<TranslatorListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportingSheet, setExportingSheet] = useState(false);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [detailRecord, setDetailRecord] = useState<CheckinItem | null>(null);
  const { message } = App.useApp();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getAdminCheckins(filters);
      setData(Array.isArray(list) ? list : []);
    } catch {
      message.error('載入打卡紀錄失敗');
    } finally {
      setLoading(false);
    }
  }, [filters, message]);

  useEffect(() => {
    void fetchData();
    void getTranslators().then(setTranslators).catch(() => undefined);
  }, [fetchData]);

  const handleExport = async () => {
    setExporting(true);
    try {
      await exportCheckinExcel(filters);
      message.success('匯出成功');
    } catch {
      message.error('匯出失敗');
    } finally {
      setExporting(false);
    }
  };

  const handleGoogleSheet = async () => {
    setExportingSheet(true);
    try {
      const res = await exportCheckinGoogleSheet(filters);
      message.success('Google Sheet 建立成功');
      window.open(res.url, '_blank');
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      if (msg?.includes('not configured')) {
        message.warning('尚未設定 Google 憑證，請聯絡系統管理員');
      } else {
        message.error('Google Sheet 匯出失敗');
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
      title: '打卡時間',
      dataIndex: 'checkinTime',
      key: 'checkinTime',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString('zh-TW'),
    },
    { title: '翻譯員', dataIndex: 'translatorName', key: 'translatorName', width: 100 },
    {
      title: '類型',
      dataIndex: 'type',
      key: 'type',
      width: 80,
      render: (t: string) => {
        const info = typeTagMap[t] ?? { color: 'default', label: t };
        return <Tag color={info.color}>{info.label}</Tag>;
      },
    },
    { title: '地址', dataIndex: 'address', key: 'address' },
    {
      title: '補打卡',
      dataIndex: 'isMakeup',
      key: 'isMakeup',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="orange">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_: unknown, record: CheckinItem) => (
        <Button size="small" onClick={() => setDetailRecord(record)}>
          詳情
        </Button>
      ),
    },
  ];

  const translatorOptions = translators.map((t) => ({ value: String(t.id), label: t.name }));

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
        <RangePicker onChange={handleDateChange} placeholder={['開始日期', '結束日期']} />
        <Select
          style={{ width: 140 }}
          allowClear
          placeholder="翻譯員"
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
          style={{ width: 110 }}
          allowClear
          placeholder="類型"
          options={[
            { value: 'arrive', label: '到達' },
            { value: 'leave', label: '離開' },
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
          style={{ width: 110 }}
          allowClear
          placeholder="補打卡"
          options={[
            { value: 'true', label: '是' },
            { value: 'false', label: '否' },
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
        <Button
          icon={<FileTextOutlined />}
          loading={exportingSheet}
          onClick={handleGoogleSheet}
        >
          匯出 Google Sheet
        </Button>
        <Button
          icon={<DownloadOutlined />}
          loading={exporting}
          onClick={handleExport}
        >
          匯出 Excel
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
        title="打卡詳情"
        open={!!detailRecord}
        onCancel={() => setDetailRecord(null)}
        footer={null}
        width={600}
      >
        {detailRecord && (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Typography.Text type="secondary">翻譯員</Typography.Text>
              <div>{detailRecord.translatorName}（ID: {detailRecord.translatorId}）</div>
            </div>
            <div>
              <Typography.Text type="secondary">打卡類型</Typography.Text>
              <div>
                <Tag color={typeTagMap[detailRecord.type]?.color}>
                  {typeTagMap[detailRecord.type]?.label}
                </Tag>
                {detailRecord.isMakeup && <Tag color="orange">補打卡</Tag>}
              </div>
            </div>
            <div>
              <Typography.Text type="secondary">打卡時間</Typography.Text>
              <div>{new Date(detailRecord.checkinTime).toLocaleString('zh-TW')}</div>
            </div>
            <div>
              <Typography.Text type="secondary">GPS 地址</Typography.Text>
              <div>{detailRecord.address || '—'}</div>
            </div>
            <div>
              <Typography.Text type="secondary">GPS 座標</Typography.Text>
              <div>{detailRecord.latitude}, {detailRecord.longitude}</div>
            </div>
            {detailRecord.isMakeup && (
              <div>
                <Typography.Text type="secondary">補打卡原因</Typography.Text>
                <div>{detailRecord.makeupReason}</div>
              </div>
            )}
            <div>
              <Typography.Text type="secondary">照片</Typography.Text>
              <div style={{ display: 'flex', gap: 12, marginTop: 8 }}>
                <div>
                  <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>自拍</div>
                  <Image
                    src={`http://localhost:8080${detailRecord.selfieUrl}`}
                    width={200}
                    style={{ borderRadius: 8 }}
                    fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
                  />
                </div>
                <div>
                  <div style={{ fontSize: 12, color: '#999', marginBottom: 4 }}>環境照</div>
                  <Image
                    src={`http://localhost:8080${detailRecord.environmentUrl}`}
                    width={200}
                    style={{ borderRadius: 8 }}
                    fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
                  />
                </div>
              </div>
            </div>
          </Space>
        )}
      </Modal>
    </div>
  );
}
