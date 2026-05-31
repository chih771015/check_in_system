import { useCallback, useEffect, useState } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, App } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { TranslatorListItem } from '../../types';
import {
  getTranslators,
  createTranslator,
  updateTranslator,
  disableTranslator,
  resetTranslatorPassword,
} from '../../api/translators';

export default function TranslatorManagement() {
  const [data, setData] = useState<TranslatorListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<TranslatorListItem | null>(null);
  const [resetOpen, setResetOpen] = useState(false);
  const [resetTarget, setResetTarget] = useState<TranslatorListItem | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [resetForm] = Form.useForm();
  // Re-entrancy guards across all three modals
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [editSubmitting, setEditSubmitting] = useState(false);
  const [resetSubmitting, setResetSubmitting] = useState(false);
  const { message, modal } = App.useApp();
  const { t } = useTranslation();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getTranslators(statusFilter || undefined);
      setData(list);
    } catch {
      message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [statusFilter, message, t]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const handleCreate = async (values: {
    name: string;
    email: string;
    phone: string;
    password: string;
  }) => {
    if (createSubmitting) return;
    setCreateSubmitting(true);
    try {
      await createTranslator(values);
      message.success(t('common.success'));
      setCreateOpen(false);
      createForm.resetFields();
      void fetchData();
    } catch {
      message.error(t('common.failed'));
    } finally {
      setCreateSubmitting(false);
    }
  };

  const handleEdit = async (values: { name: string; phone: string; status: string }) => {
    if (!editingRecord) return;
    if (editSubmitting) return;
    setEditSubmitting(true);
    try {
      await updateTranslator(editingRecord.id, values);
      message.success(t('common.success'));
      setEditOpen(false);
      void fetchData();
    } catch {
      message.error(t('common.failed'));
    } finally {
      setEditSubmitting(false);
    }
  };

  const handleDisable = (record: TranslatorListItem) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('translators.confirmDisable', { name: record.name }),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await disableTranslator(record.id);
          message.success(t('common.success'));
          void fetchData();
        } catch {
          message.error(t('common.failed'));
        }
      },
    });
  };

  const openEdit = (record: TranslatorListItem) => {
    setEditingRecord(record);
    editForm.setFieldsValue({ name: record.name, phone: record.phone, status: record.status });
    setEditOpen(true);
  };

  const openReset = (record: TranslatorListItem) => {
    setResetTarget(record);
    resetForm.resetFields();
    setResetOpen(true);
  };

  const handleReset = async (values: { newPassword: string; confirmPassword: string }) => {
    if (!resetTarget) return;
    if (resetSubmitting) return;
    if (values.newPassword !== values.confirmPassword) {
      message.error(t('changePassword.mismatch'));
      return;
    }
    setResetSubmitting(true);
    try {
      await resetTranslatorPassword(resetTarget.id, values.newPassword);
      message.success(t('common.success'));
      setResetOpen(false);
    } catch {
      message.error(t('common.failed'));
    } finally {
      setResetSubmitting(false);
    }
  };

  const columns = [
    { title: t('common.id'), dataIndex: 'id', key: 'id', width: 60 },
    { title: t('common.name'), dataIndex: 'name', key: 'name' },
    { title: t('common.email'), dataIndex: 'email', key: 'email' },
    { title: t('common.phone'), dataIndex: 'phone', key: 'phone' },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) =>
        status === 'active' ? <Tag color="green">{t('common.active')}</Tag> : <Tag color="red">{t('common.disabled')}</Tag>,
    },
    {
      title: t('common.operation'),
      key: 'action',
      render: (_: unknown, record: TranslatorListItem) => (
        <Space wrap>
          <Button size="small" onClick={() => openEdit(record)}>
            {t('common.edit')}
          </Button>
          <Button size="small" onClick={() => openReset(record)}>
            {t('translators.resetPassword')}
          </Button>
          {record.status === 'active' && (
            <Button size="small" danger onClick={() => handleDisable(record)}>
              {t('translators.disable')}
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
          style={{ width: 140 }}
          value={statusFilter}
          onChange={setStatusFilter}
          options={[
            { value: '', label: t('common.actions') },
            { value: 'active', label: t('common.active') },
            { value: 'disabled', label: t('common.disabled') },
          ]}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          {t('translators.add')}
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
        title={t('translators.add')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
      >
        <Form form={createForm} onFinish={handleCreate} layout="vertical">
          <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="email"
            label={t('common.email')}
            rules={[{ required: true }, { type: 'email' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="phone" label={t('common.phone')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="password"
            label={t('common.password')}
            rules={[{ required: true }, { min: 6 }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={createSubmitting} disabled={createSubmitting}>
              {t('common.create')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('translators.edit')} open={editOpen} onCancel={() => setEditOpen(false)} footer={null}>
        <Form form={editForm} onFinish={handleEdit} layout="vertical">
          <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="phone" label={t('common.phone')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="status" label={t('common.status')}>
            <Select
              options={[
                { value: 'active', label: t('common.active') },
                { value: 'disabled', label: t('common.disabled') },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={editSubmitting} disabled={editSubmitting}>
              {t('common.update')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('translators.resetPassword')}
        open={resetOpen}
        onCancel={() => setResetOpen(false)}
        footer={null}
      >
        <Form form={resetForm} onFinish={handleReset} layout="vertical">
          <Form.Item
            name="newPassword"
            label={t('common.newPassword')}
            rules={[{ required: true }, { min: 8, message: t('changePassword.minLength') }]}
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
            <Button type="primary" htmlType="submit" block loading={resetSubmitting} disabled={resetSubmitting}>
              {t('translators.resetPassword')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
