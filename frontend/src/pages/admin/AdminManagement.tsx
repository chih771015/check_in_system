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
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getAdmins();
      setData(list);
    } catch {
      void message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const handleCreate = async (values: { email: string; name: string; password: string; confirmPassword: string }) => {
    if (values.password !== values.confirmPassword) {
      void message.error(t('changePassword.mismatch'));
      return;
    }
    setCreateLoading(true);
    try {
      await createAdmin({ email: values.email, name: values.name, password: values.password });
      void message.success(t('admins.createSuccess'));
      setCreateOpen(false);
      createForm.resetFields();
      void fetchData();
    } catch {
      void message.error(t('admins.createFailed'));
    } finally {
      setCreateLoading(false);
    }
  };

  const handleDelete = (record: AdminListItem) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('admins.confirmDelete', { name: record.name, email: record.email }),
      okText: t('common.delete'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deleteAdmin(record.id);
          void message.success(t('common.success'));
          void fetchData();
        } catch {
          void message.error(t('common.failed'));
        }
      },
    });
  };

  const columns = [
    { title: t('common.id'), dataIndex: 'id', key: 'id', width: 60 },
    { title: t('common.name'), dataIndex: 'name', key: 'name' },
    { title: t('common.email'), dataIndex: 'email', key: 'email' },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (v: string) =>
        v === 'active' ? <Tag color="green">{t('common.active')}</Tag> : <Tag color="red">{t('common.disabled')}</Tag>,
    },
    {
      title: t('common.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: t('common.operation'),
      key: 'action',
      width: 100,
      render: (_: unknown, record: AdminListItem) => {
        const isSelf = record.id === user?.id;
        return (
          <Space>
            <Tooltip title={isSelf ? t('admins.deleteSelfTooltip') : ''}>
              <Button
                size="small"
                danger
                disabled={isSelf}
                onClick={() => handleDelete(record)}
              >
                {t('common.delete')}
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
        <Typography.Title level={4} style={{ margin: 0 }}>{t('admins.title')}</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          {t('admins.add')}
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
        title={t('admins.add')}
        open={createOpen}
        onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        footer={null}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={handleCreate} style={{ marginTop: 8 }}>
          <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="email"
            label={t('common.email')}
            rules={[
              { required: true },
              { type: 'email' },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="password"
            label={t('common.password')}
            rules={[
              { required: true },
              { min: 8, message: t('changePassword.minLength') },
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={t('common.confirmPassword')}
            rules={[{ required: true }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={createLoading}>
              {t('common.create')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
