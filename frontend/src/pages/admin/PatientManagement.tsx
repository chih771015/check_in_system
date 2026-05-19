import { useCallback, useEffect, useState } from 'react';
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
} from 'antd';
import { PlusOutlined, SearchOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { IDType, Patient } from '../../types';
import {
  createPatient,
  deletePatient,
  getPatients,
  updatePatient,
  type PatientPayload,
} from '../../api/patients';

const ID_TYPE_LABEL: Record<IDType, string> = {
  passport: '護照',
  hn: '病歷號 (HN)',
  unid: '識別號 (UNID)',
};

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

  const { message, modal } = App.useApp();
  const navigate = useNavigate();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getPatients({ search, page, pageSize: PAGE_SIZE });
      setData(res.data);
      setTotal(res.total);
    } catch {
      void message.error('載入病人列表失敗');
    } finally {
      setLoading(false);
    }
  }, [search, page, message]);

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
        void message.success('病人資料已更新');
      } else {
        await createPatient(payload);
        void message.success('已新增病人資料');
      }
      setModalOpen(false);
      void fetchData();
    } catch (err: unknown) {
      const msg = (err as { response?: { status?: number } })?.response?.status === 409
        ? '此 ID 類型與號碼已存在，請勿重複建立'
        : '操作失敗';
      void message.error(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = (record: Patient) => {
    modal.confirm({
      title: '確認刪除',
      content: `確定要刪除病人「${record.name}」嗎？此操作無法復原。`,
      okText: '確認刪除',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deletePatient(record.id);
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
    { title: '電話', dataIndex: 'phone', key: 'phone' },
    {
      title: 'ID 類型',
      dataIndex: 'idType',
      key: 'idType',
      width: 130,
      render: (v: IDType) => <Tag color={ID_TYPE_COLOR[v]}>{ID_TYPE_LABEL[v]}</Tag>,
    },
    { title: 'ID 號碼', dataIndex: 'idNumber', key: 'idNumber' },
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
      width: 230,
      render: (_: unknown, record: Patient) => (
        <Space>
          <Button size="small" onClick={() => navigate(`/admin/patients/${record.id}/history`)}>
            查看歷史
          </Button>
          <Button size="small" onClick={() => openEdit(record)}>編輯</Button>
          <Button size="small" danger onClick={() => handleDelete(record)}>
            刪除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <Typography.Title level={4} style={{ margin: 0 }}>病人資料管理</Typography.Title>
        <Space>
          <Input
            allowClear
            placeholder="搜尋姓名 / 電話 / ID 號碼"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onPressEnter={() => { setPage(1); setSearch(searchInput.trim()); }}
            style={{ width: 260 }}
            prefix={<SearchOutlined />}
          />
          <Button onClick={() => { setPage(1); setSearch(searchInput.trim()); }}>搜尋</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新增病人
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
        title={editing ? '編輯病人資料' : '新增病人'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit} style={{ marginTop: 8 }}>
          <Form.Item name="name" label="姓名" rules={[{ required: true, message: '請輸入姓名' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="phone" label="電話" rules={[{ required: true, message: '請輸入電話' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="idType" label="ID 類型" rules={[{ required: true, message: '請選擇 ID 類型' }]}>
            <Select
              options={[
                { value: 'passport', label: ID_TYPE_LABEL.passport },
                { value: 'hn', label: ID_TYPE_LABEL.hn },
                { value: 'unid', label: ID_TYPE_LABEL.unid },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="idNumber"
            label="ID 號碼"
            rules={[{ required: true, message: '請輸入 ID 號碼' }]}
            extra="自動轉為大寫存入資料庫"
          >
            <Input />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={submitting}>
              {editing ? '儲存變更' : '建立病人'}
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
