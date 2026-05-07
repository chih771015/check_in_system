import { useCallback, useEffect, useState } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  DatePicker,
  TimePicker,
  Tag,
  Space,
  App,
} from 'antd';
import { PlusOutlined, UploadOutlined, DownloadOutlined } from '@ant-design/icons';
import { Upload } from 'antd';
import type { UploadProps } from 'antd';
import type { ScheduleItem, TranslatorListItem } from '../../types';
import {
  getAdminSchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
  deleteScheduleGroup,
  importSchedules,
} from '../../api/schedules';
import { getTranslators } from '../../api/translators';
import * as XLSX from 'xlsx';

function downloadImportTemplate() {
  const headers = ['翻譯員ID', '日期(YYYY-MM-DD)', '開始時間(HH:mm)', '結束時間(HH:mm)', '地點', '病患姓名', '備註(選填)'];
  const example = [3, '2026-05-10', '09:00', '12:00', '台大醫院門診', '王小明', ''];
  const ws = XLSX.utils.aoa_to_sheet([headers, example]);

  // 欄位寬度
  ws['!cols'] = [
    { wch: 12 }, { wch: 18 }, { wch: 16 }, { wch: 16 },
    { wch: 20 }, { wch: 12 }, { wch: 16 },
  ];

  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, '排班匯入範本');
  XLSX.writeFile(wb, '排班匯入範本.xlsx');
}

const { RangePicker } = DatePicker;

const statusTagMap: Record<string, { color: string; label: string }> = {
  none: { color: 'default', label: '未打卡' },
  arrived: { color: 'orange', label: '已到達' },
  completed: { color: 'green', label: '已完成' },
  makeup: { color: 'blue', label: '補打卡' },
};

export default function ScheduleManagement() {
  const [data, setData] = useState<ScheduleItem[]>([]);
  const [translators, setTranslators] = useState<TranslatorListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<ScheduleItem | null>(null);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const { message, modal } = App.useApp();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getAdminSchedules(filters);
      setData(list);
    } catch {
      message.error('載入排班列表失敗');
    } finally {
      setLoading(false);
    }
  }, [filters, message]);

  const fetchTranslators = useCallback(async () => {
    try {
      const list = await getTranslators('active');
      setTranslators(list);
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    void fetchData();
    void fetchTranslators();
  }, [fetchData, fetchTranslators]);

  const handleCreate = async (values: Record<string, unknown>) => {
    try {
      const payload: Record<string, unknown> = {
        translatorId: values.translatorId as number,
        date: (values.date as { format: (f: string) => string }).format('YYYY-MM-DD'),
        startTime: (values.startTime as { format: (f: string) => string }).format('HH:mm'),
        endTime: (values.endTime as { format: (f: string) => string }).format('HH:mm'),
        location: values.location as string,
        patientName: values.patientName as string,
        note: (values.note as string) || undefined,
      };
      if (values.recurrenceRule) {
        payload.recurrenceRule = values.recurrenceRule as string;
        payload.recurrenceUntil = (values.recurrenceUntil as { format: (f: string) => string }).format('YYYY-MM-DD');
      }
      await createSchedule(payload as Parameters<typeof createSchedule>[0]);
      message.success('新增成功');
      setCreateOpen(false);
      createForm.resetFields();
      void fetchData();
    } catch {
      message.error('新增失敗');
    }
  };

  const handleEdit = async (values: Record<string, unknown>) => {
    if (!editingRecord) return;
    try {
      const payload = {
        translatorId: values.translatorId as number,
        date: (values.date as { format: (f: string) => string }).format('YYYY-MM-DD'),
        startTime: (values.startTime as { format: (f: string) => string }).format('HH:mm'),
        endTime: (values.endTime as { format: (f: string) => string }).format('HH:mm'),
        location: values.location as string,
        patientName: values.patientName as string,
        note: (values.note as string) || undefined,
      };
      await updateSchedule(editingRecord.id, payload);
      message.success('更新成功');
      setEditOpen(false);
      void fetchData();
    } catch {
      message.error('更新失敗');
    }
  };

  const handleDelete = (record: ScheduleItem) => {
    modal.confirm({
      title: '確認刪除',
      content: `確定要刪除此排班嗎？`,
      okText: '確認',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deleteSchedule(record.id);
          message.success('已刪除');
          void fetchData();
        } catch {
          message.error('刪除失敗');
        }
      },
    });
  };

  const handleDeleteGroup = (record: ScheduleItem) => {
    modal.confirm({
      title: '刪除整組重複排班',
      content: `確定要刪除此排班所屬的整組重複排班嗎？此操作無法復原。`,
      okText: '刪除整組',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          const res = await deleteScheduleGroup(record.id);
          message.success(`已刪除整組 ${res?.deleted ?? ''} 筆排班`);
          void fetchData();
        } catch {
          message.error('刪除整組失敗');
        }
      },
    });
  };

  const openEdit = (record: ScheduleItem) => {
    setEditingRecord(record);
    editForm.setFieldsValue({
      translatorId: record.translatorId,
      location: record.location,
      patientName: record.patientName,
      note: record.note,
    });
    setEditOpen(true);
  };

  const handleDateRangeChange = (_: unknown, dateStrings: [string, string]) => {
    if (dateStrings[0] && dateStrings[1]) {
      setFilters((prev) => ({ ...prev, startDate: dateStrings[0], endDate: dateStrings[1] }));
    } else {
      setFilters((prev) => {
        const next = { ...prev };
        delete next.startDate;
        delete next.endDate;
        return next;
      });
    }
  };

  const handleTranslatorFilter = (value: string) => {
    if (value) {
      setFilters((prev) => ({ ...prev, translatorId: value }));
    } else {
      setFilters((prev) => {
        const next = { ...prev };
        delete next.translatorId;
        return next;
      });
    }
  };

  const handleLocationSearch = (value: string) => {
    if (value) {
      setFilters((prev) => ({ ...prev, location: value }));
    } else {
      setFilters((prev) => {
        const next = { ...prev };
        delete next.location;
        return next;
      });
    }
  };

  const columns = [
    { title: '日期', dataIndex: 'date', key: 'date', width: 110 },
    {
      title: '時間',
      key: 'time',
      width: 120,
      render: (_: unknown, r: ScheduleItem) => `${r.startTime} - ${r.endTime}`,
    },
    { title: '翻譯員', dataIndex: 'translatorName', key: 'translatorName' },
    { title: '地點', dataIndex: 'location', key: 'location' },
    { title: '病患姓名', dataIndex: 'patientName', key: 'patientName' },
    {
      title: '打卡狀態',
      dataIndex: 'checkinStatus',
      key: 'checkinStatus',
      render: (status: string) => {
        const info = statusTagMap[status] ?? statusTagMap.none;
        return <Tag color={info.color}>{info.label}</Tag>;
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: ScheduleItem) => (
        <Space wrap>
          <Button size="small" onClick={() => openEdit(record)}>
            編輯
          </Button>
          <Button size="small" danger onClick={() => handleDelete(record)}>
            刪除
          </Button>
          {record.recurrenceGroupId && (
            <Button size="small" danger onClick={() => handleDeleteGroup(record)}>
              刪除整組
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const translatorOptions = translators.map((t) => ({ value: t.id, label: t.name }));

  const scheduleFormFields = (
    <>
      <Form.Item
        name="translatorId"
        label="翻譯員"
        rules={[{ required: true, message: '請選擇翻譯員' }]}
      >
        <Select options={translatorOptions} placeholder="請選擇" showSearch optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="date" label="日期" rules={[{ required: true, message: '請選擇日期' }]}>
        <DatePicker style={{ width: '100%' }} />
      </Form.Item>
      <Space style={{ width: '100%' }} size="middle">
        <Form.Item
          name="startTime"
          label="開始時間"
          rules={[{ required: true, message: '請選擇開始時間' }]}
        >
          <TimePicker format="HH:mm" />
        </Form.Item>
        <Form.Item
          name="endTime"
          label="結束時間"
          rules={[{ required: true, message: '請選擇結束時間' }]}
        >
          <TimePicker format="HH:mm" />
        </Form.Item>
      </Space>
      <Form.Item
        name="location"
        label="地點"
        rules={[{ required: true, message: '請輸入地點' }]}
      >
        <Input />
      </Form.Item>
      <Form.Item
        name="patientName"
        label="病患姓名"
        rules={[{ required: true, message: '請輸入病患姓名' }]}
      >
        <Input />
      </Form.Item>
      <Form.Item name="note" label="備註">
        <Input.TextArea rows={2} />
      </Form.Item>
    </>
  );

  const recurrenceFields = (
    <>
      <Form.Item name="recurrenceRule" label="重複規則">
        <Select
          allowClear
          placeholder="不重複"
          options={[
            { value: 'daily', label: '每日' },
            { value: 'weekly:1,3,5', label: '每週一三五' },
            { value: 'weekly:2,4', label: '每週二四' },
            { value: 'weekly:1,2,3,4,5', label: '每週一至五' },
            { value: 'monthly:1', label: '每月1日' },
            { value: 'monthly:15', label: '每月15日' },
          ]}
        />
      </Form.Item>
      <Form.Item
        noStyle
        shouldUpdate={(prev, curr) => prev.recurrenceRule !== curr.recurrenceRule}
      >
        {({ getFieldValue }) =>
          getFieldValue('recurrenceRule') ? (
            <Form.Item
              name="recurrenceUntil"
              label="重複至"
              rules={[{ required: true, message: '請選擇結束日期' }]}
            >
              <DatePicker style={{ width: '100%' }} placeholder="重複結束日期" />
            </Form.Item>
          ) : null
        }
      </Form.Item>
    </>
  );

  return (
    <div>
      <div
        style={{
          marginBottom: 16,
          display: 'flex',
          flexWrap: 'wrap',
          gap: 8,
          alignItems: 'center',
        }}
      >
        <RangePicker onChange={handleDateRangeChange} placeholder={['開始日期', '結束日期']} />
        <Select
          style={{ width: 160 }}
          allowClear
          placeholder="翻譯員"
          options={translatorOptions}
          onChange={(v) => handleTranslatorFilter(v ? String(v) : '')}
          showSearch
          optionFilterProp="label"
        />
        <Input.Search
          style={{ width: 200 }}
          placeholder="搜尋地點"
          allowClear
          onSearch={handleLocationSearch}
        />
        <div style={{ flex: 1 }} />
        <Button icon={<DownloadOutlined />} onClick={downloadImportTemplate}>
          下載匯入範本
        </Button>
        <Upload
          {...({
            accept: '.xlsx,.xls',
            showUploadList: false,
            beforeUpload: (file: File) => {
              importSchedules(file)
                .then((res) => {
                  if (res.failed && res.failed.length > 0) {
                    message.warning(
                      `成功 ${res.success} 筆，失敗 ${res.failed.length} 筆（第一筆錯誤：${res.failed[0].error}）`,
                    );
                  } else {
                    message.success(`匯入成功 ${res.success} 筆`);
                  }
                  void fetchData();
                })
                .catch(() => message.error('匯入失敗'));
              return false;
            },
          } as UploadProps)}
        >
          <Button icon={<UploadOutlined />}>Excel 批次匯入</Button>
        </Upload>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新增排班
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        scroll={{ x: 800 }}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title="新增排班"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto' } }}
      >
        <Form form={createForm} onFinish={handleCreate} layout="vertical">
          {scheduleFormFields}
          {recurrenceFields}
          <Form.Item>
            <Button type="primary" htmlType="submit" block>
              新增
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="編輯排班" open={editOpen} onCancel={() => setEditOpen(false)} footer={null}>
        <Form form={editForm} onFinish={handleEdit} layout="vertical">
          {scheduleFormFields}
          <Form.Item>
            <Button type="primary" htmlType="submit" block>
              更新
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
