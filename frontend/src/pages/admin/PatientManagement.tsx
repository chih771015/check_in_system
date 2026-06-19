import { useCallback, useEffect, useRef, useState } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Space,
  Typography,
  App,
  Tag,
  Tooltip,
} from 'antd';
import { PlusOutlined, SearchOutlined, UploadOutlined, DownloadOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { IDType, Patient } from '../../types';
import {
  createPatient,
  deletePatient,
  getPatients,
  updatePatient,
  importPatients,
  exportPatients,
  downloadPatientTemplate,
  type PatientPayload,
} from '../../api/patients';
import { extractApiError } from '../../utils/apiError';

const ID_TYPE_COLOR: Record<IDType, string> = {
  passport: 'blue',
  hn: 'green',
  unid: 'orange',
};

const PAGE_SIZE = 20;

export default function PatientManagement() {
  const [data, setData] = useState<Patient[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [loading, setLoading] = useState(false);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<Patient | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<PatientPayload>();

  const [importing, setImporting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { message, modal } = App.useApp();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getPatients({ search, page, pageSize: PAGE_SIZE });
      setData(res.data);
      setTotal(res.total);
    } catch {
      void message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [search, page, message, t]);

  useEffect(() => {
    void fetchData();
  }, [fetchData]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const openEdit = (record: Patient) => {
    setEditing(record);
    form.setFieldsValue({
      name: record.name,
      phone: record.phone,
      idType: record.idType,
      idNumber: record.idNumber,
    });
    setModalOpen(true);
  };

  const handleSubmit = async (values: PatientPayload) => {
    setSubmitting(true);
    try {
      const payload: PatientPayload = {
        ...values,
        idNumber: values.idNumber.trim().toUpperCase(),
      };
      if (editing) {
        await updatePatient(editing.id, payload);
        void message.success(t('common.success'));
      } else {
        await createPatient(payload);
        void message.success(t('common.success'));
      }
      setModalOpen(false);
      void fetchData();
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      void message.error(status === 409 ? t('errors.PATIENT_DUPLICATE') : t('common.failed'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = (record: Patient) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('patients.confirmDelete', { name: record.name }),
      okText: t('common.delete'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deletePatient(record.id);
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
    { title: t('common.phone'), dataIndex: 'phone', key: 'phone' },
    {
      title: t('patients.idType'),
      dataIndex: 'idType',
      key: 'idType',
      width: 130,
      render: (v: IDType) => <Tag color={ID_TYPE_COLOR[v]}>{t(`patients.idTypes.${v}`)}</Tag>,
    },
    { title: t('patients.idNumber'), dataIndex: 'idNumber', key: 'idNumber' },
    {
      title: t('patients.actualTotal'),
      dataIndex: 'actualTotal',
      key: 'actualTotal',
      width: 130,
      render: (v: number | undefined) => `NT$ ${(v ?? 0).toLocaleString()}`,
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
      width: 230,
      render: (_: unknown, record: Patient) => (
        <Space>
          <Button size="small" onClick={() => navigate(`/admin/patients/${record.id}/history`)}>
            {t('patients.history')}
          </Button>
          <Button size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
          <Button size="small" danger onClick={() => handleDelete(record)}>
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = ''; // reset so the same file can be re-picked
    if (!file) return;
    setImporting(true);
    try {
      const res = await importPatients(file);
      void message.success(t('patients.importResult', { created: res.created, skipped: res.skipped }));
      if (res.errors.length > 0) {
        modal.info({
          title: t('patients.importErrorsTitle'),
          content: (
            <div style={{ maxHeight: 300, overflow: 'auto' }}>
              {res.errors.slice(0, 50).map((err) => (
                <div key={err.row} style={{ fontSize: 13 }}>
                  {t('patients.importErrorRow', { row: err.row })}: {err.reason}
                </div>
              ))}
            </div>
          ),
        });
      }
      void fetchData();
    } catch (err: unknown) {
      void message.error(extractApiError(err) || t('common.failed'));
    } finally {
      setImporting(false);
    }
  };

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <Typography.Title level={4} style={{ margin: 0 }}>{t('patients.title')}</Typography.Title>
        <Space wrap>
          <Input
            allowClear
            placeholder={t('patients.searchPlaceholder')}
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onPressEnter={() => { setPage(1); setSearch(searchInput.trim()); }}
            style={{ width: 260 }}
            prefix={<SearchOutlined />}
          />
          <Button onClick={() => { setPage(1); setSearch(searchInput.trim()); }}>{t('common.search')}</Button>
          <Button icon={<DownloadOutlined />} onClick={() => void downloadPatientTemplate()}>
            {t('patients.downloadTemplate')}
          </Button>
          <Tooltip title={t('patients.importHint')}>
            <Button icon={<UploadOutlined />} loading={importing} onClick={() => fileInputRef.current?.click()}>
              {t('patients.import')}
            </Button>
          </Tooltip>
          <Button icon={<DownloadOutlined />} onClick={() => void exportPatients()}>
            {t('patients.export')}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".xlsx"
            style={{ display: 'none' }}
            onChange={handleImportFile}
          />
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            {t('patients.add')}
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize: PAGE_SIZE,
          total,
          showSizeChanger: false,
          onChange: (p) => setPage(p),
        }}
      />

      <Modal
        title={editing ? t('patients.edit') : t('patients.add')}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} style={{ marginTop: 8 }}>
          <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="phone" label={t('common.phone')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="idType" label={t('patients.idType')} rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'passport', label: t('patients.idTypes.passport') },
                { value: 'hn', label: t('patients.idTypes.hn') },
                { value: 'unid', label: t('patients.idTypes.unid') },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="idNumber"
            label={t('patients.idNumber')}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={submitting}>
              {editing ? t('common.save') : t('common.create')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
