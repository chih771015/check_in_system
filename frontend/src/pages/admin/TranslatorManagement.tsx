import { useCallback, useEffect, useState } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, App } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { TranslatorListItem } from '../../types';
import {
  getTranslators,
  createTranslator,
  updateTranslator,
  disableTranslator,
} from '../../api/translators';

export default function TranslatorManagement() {
  const [data, setData] = useState<TranslatorListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<TranslatorListItem | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const { message, modal } = App.useApp();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getTranslators(statusFilter || undefined);
      setData(list);
    } catch {
      message.error('載入翻譯員列表失敗');
    } finally {
      setLoading(false);
    }
  }, [statusFilter, message]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const handleCreate = async (values: {
    name: string;
    email: string;
    phone: string;
    password: string;
  }) => {
    try {
      await createTranslator(values);
      message.success('新增成功');
      setCreateOpen(false);
      createForm.resetFields();
      void fetchData();
    } catch {
      message.error('新增失敗');
    }
  };

  const handleEdit = async (values: { name: string; phone: string; status: string }) => {
    if (!editingRecord) return;
    try {
      await updateTranslator(editingRecord.id, values);
      message.success('更新成功');
      setEditOpen(false);
      void fetchData();
    } catch {
      message.error('更新失敗');
    }
  };

  const handleDisable = (record: TranslatorListItem) => {
    modal.confirm({
      title: '確認停用',
      content: `確定要停用翻譯員「${record.name}」嗎？`,
      okText: '確認',
      cancelText: '取消',
      onOk: async () => {
        try {
          await disableTranslator(record.id);
          message.success('已停用');
          void fetchData();
        } catch {
          message.error('停用失敗');
        }
      },
    });
  };

  const openEdit = (record: TranslatorListItem) => {
    setEditingRecord(record);
    editForm.setFieldsValue({ name: record.name, phone: record.phone, status: record.status });
    setEditOpen(true);
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '姓名', dataIndex: 'name', key: 'name' },
    { title: '電子信箱', dataIndex: 'email', key: 'email' },
    { title: '電話', dataIndex: 'phone', key: 'phone' },
    {
      title: '狀態',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) =>
        status === 'active' ? <Tag color="green">啟用</Tag> : <Tag color="red">停用</Tag>,
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: TranslatorListItem) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)}>
            編輯
          </Button>
          {record.status === 'active' && (
            <Button size="small" danger onClick={() => handleDisable(record)}>
              停用
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div
        style={{
          marginBottom: 16,
          display: 'flex',
          justifyContent: 'space-between',
          flexWrap: 'wrap',
          gap: 8,
        }}
      >
        <Select
          style={{ width: 120 }}
          value={statusFilter}
          onChange={setStatusFilter}
          options={[
            { value: '', label: '全部' },
            { value: 'active', label: '啟用' },
            { value: 'disabled', label: '停用' },
          ]}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新增翻譯員
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        scroll={{ x: 600 }}
        pagination={{ pageSize: 10 }}
      />

      <Modal
        title="新增翻譯員"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
      >
        <Form form={createForm} onFinish={handleCreate} layout="vertical">
          <Form.Item name="name" label="姓名" rules={[{ required: true, message: '請輸入姓名' }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="email"
            label="電子信箱"
            rules={[
              { required: true, message: '請輸入電子信箱' },
              { type: 'email', message: '請輸入正確的信箱格式' },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="phone" label="電話" rules={[{ required: true, message: '請輸入電話' }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="password"
            label="密碼"
            rules={[
              { required: true, message: '請輸入密碼' },
              { min: 6, message: '密碼至少6個字元' },
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block>
              新增
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="編輯翻譯員" open={editOpen} onCancel={() => setEditOpen(false)} footer={null}>
        <Form form={editForm} onFinish={handleEdit} layout="vertical">
          <Form.Item name="name" label="姓名" rules={[{ required: true, message: '請輸入姓名' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="phone" label="電話" rules={[{ required: true, message: '請輸入電話' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="status" label="狀態">
            <Select
              options={[
                { value: 'active', label: '啟用' },
                { value: 'disabled', label: '停用' },
              ]}
            />
          </Form.Item>
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
