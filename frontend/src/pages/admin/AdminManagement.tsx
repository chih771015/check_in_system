import { useCallback, useEffect, useState } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Tag,
  Space,
  Tooltip,
  Typography,
  App,
} from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import type { AdminListItem } from '../../types';
import { getAdmins, createAdmin, deleteAdmin } from '../../api/admins';
import { useAuth } from '../../stores/authStore';

export default function AdminManagement() {
  const [data, setData] = useState<AdminListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [createForm] = Form.useForm();
  const { user } = useAuth();
  const { message, modal } = App.useApp();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getAdmins();
      setData(list);
    } catch {
      void message.error('載入管理員列表失敗');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const handleCreate = async (values: { email: string; name: string; password: string; confirmPassword: string }) => {
    if (values.password !== values.confirmPassword) {
      void message.error('兩次輸入的密碼不一致');
      return;
    }
    setCreateLoading(true);
    try {
      await createAdmin({ email: values.email, name: values.name, password: values.password });
      void message.success('管理員帳號已建立，首次登入需修改密碼');
      setCreateOpen(false);
      createForm.resetFields();
      void fetchData();
    } catch {
      void message.error('建立失敗，Email 可能已被使用');
    } finally {
      setCreateLoading(false);
    }
  };

  const handleDelete = (record: AdminListItem) => {
    modal.confirm({
      title: '確認刪除',
      content: `確定要刪除管理員「${record.name}」（${record.email}）嗎？此操作無法復原。`,
      okText: '確認刪除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deleteAdmin(record.id);
          void message.success('已刪除');
          void fetchData();
        } catch {
          void message.error('刪除失敗');
        }
      },
    });
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '姓名', dataIndex: 'name', key: 'name' },
    { title: 'Email', dataIndex: 'email', key: 'email' },
    {
      title: '狀態',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (v: string) =>
        v === 'active' ? <Tag color="green">啟用</Tag> : <Tag color="red">停用</Tag>,
    },
    {
      title: '建立時間',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString('zh-TW'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: AdminListItem) => {
        const isSelf = record.id === user?.id;
        return (
          <Space>
            <Tooltip title={isSelf ? '無法刪除自己的帳號' : ''}>
              <Button
                size="small"
                danger
                disabled={isSelf}
                onClick={() => handleDelete(record)}
              >
                刪除
              </Button>
            </Tooltip>
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography.Title level={4} style={{ margin: 0 }}>管理員帳號管理</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          新增管理員
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={false}
      />

      <Modal
        title="新增管理員帳號"
        open={createOpen}
        onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        footer={null}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreate} style={{ marginTop: 8 }}>
          <Form.Item name="name" label="姓名" rules={[{ required: true, message: '請輸入姓名' }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="email"
            label="Email"
            rules={[
              { required: true, message: '請輸入 Email' },
              { type: 'email', message: '請輸入有效的 Email' },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="password"
            label="初始密碼"
            rules={[
              { required: true, message: '請輸入初始密碼' },
              { min: 8, message: '密碼至少 8 個字元' },
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="確認密碼"
            rules={[{ required: true, message: '請再次輸入密碼' }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={createLoading}>
              建立帳號
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
