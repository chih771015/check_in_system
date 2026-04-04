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
import { PlusOutlined } from '@ant-design/icons';
import type { ScheduleItem, TranslatorListItem } from '../../types';
import {
  getAdminSchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
} from '../../api/schedules';
import { getTranslators } from '../../api/translators';

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
      const payload = {
        translatorId: values.translatorId as number,
        date: (values.date as { format: (f: string) => string }).format('YYYY-MM-DD'),
        startTime: (values.startTime as { format: (f: string) => string }).format('HH:mm'),
        endTime: (values.endTime as { format: (f: string) => string }).format('HH:mm'),
        location: values.location as string,
        patientName: values.patientName as string,
        note: (values.note as string) || undefined,
      };
      await createSchedule(payload);
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
        <Space>
          <Button size="small" onClick={() => openEdit(record)}>
            編輯
          </Button>
          <Button size="small" danger onClick={() => handleDelete(record)}>
            刪除
          </Button>
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
      >
        <Form form={createForm} onFinish={handleCreate} layout="vertical">
          {scheduleFormFields}
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
