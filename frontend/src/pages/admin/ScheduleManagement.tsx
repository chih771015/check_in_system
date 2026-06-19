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
  Typography,
  Tooltip,
} from 'antd';
import { PlusOutlined, UploadOutlined, DownloadOutlined, ClockCircleOutlined } from '@ant-design/icons';
import { Upload } from 'antd';
import type { UploadProps } from 'antd';
import { useTranslation } from 'react-i18next';
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
import SchedulePatientListEditor from '../../components/SchedulePatientListEditor';
import DiagnosisUploadModal from '../../components/DiagnosisUploadModal';
import NoShowModal from '../../components/NoShowModal';
import {
  adminUploadDiagnosis,
  adminMarkNoShow,
  adminSetActualAmount,
  adminListDiagnosisPhotos,
  adminDeleteDiagnosisPhoto,
} from '../../api/checkins';
import { getSchedulePatientPhotos } from '../../api/diagnosisResults';
import { validatePatientTimes } from '../../utils/schedulePatient';
import { extractApiError } from '../../utils/apiError';
import type { SchedulePatientPayload, SchedulePatient } from '../../types';
import { Image } from 'antd';
import * as XLSX from 'xlsx';

const spStatusColor: Record<string, string> = {
  pending: 'orange',
  completed: 'green',
  no_show: 'red',
};

// Stage-3 V2 flat template. Rows with the same Code merge into one schedule
// with multiple patients.
function downloadImportTemplate() {
  const headers = [
    'Code', 'TranslatorID', 'Date(YYYY-MM-DD)',
    'OverallStart(HH:mm)', 'OverallEnd(HH:mm)',
    'Location', 'PatientID',
    'PatientStart(HH:mm)', 'PatientEnd(HH:mm)',
    'Note(optional)',
  ];
  const examples = [
    ['SCH-001', 3, '2026-05-10', '09:00', '12:00', 'NTU Hospital', 12, '09:00', '10:00', ''],
    ['SCH-001', 3, '2026-05-10', '09:00', '12:00', 'NTU Hospital', 15, '10:00', '11:00', ''],
    ['SCH-002', 5, '2026-05-10', '14:00', '17:00', 'VGH', 22, '14:00', '15:00', 'needs sign'],
  ];
  const ws = XLSX.utils.aoa_to_sheet([headers, ...examples]);
  ws['!cols'] = [
    { wch: 10 }, { wch: 12 }, { wch: 18 },
    { wch: 16 }, { wch: 16 },
    { wch: 20 }, { wch: 10 },
    { wch: 16 }, { wch: 16 },
    { wch: 16 },
  ];
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, 'Template');
  XLSX.writeFile(wb, 'schedule_template.xlsx');
}

const { RangePicker } = DatePicker;

const statusColorMap: Record<string, string> = {
  none: 'default',
  arrived: 'orange',
  completed: 'green',
  makeup: 'blue',
};

export default function ScheduleManagement() {
  const [data, setData] = useState<ScheduleItem[]>([]);
  const [translators, setTranslators] = useState<TranslatorListItem[]>([]);
  const [createPatients, setCreatePatients] = useState<SchedulePatientPayload[]>([]);
  const [editPatients, setEditPatients] = useState<SchedulePatientPayload[]>([]);
  // Admin surrogate: detail modal + diagnosis / no-show modal state
  const [detailRecord, setDetailRecord] = useState<ScheduleItem | null>(null);
  const [adminDiagRow, setAdminDiagRow] = useState<SchedulePatient | null>(null);
  const [adminNoShowFor, setAdminNoShowFor] = useState<number | null>(null);
  // Photo preview modal — lazy-loaded list of URLs for one SchedulePatient.
  const [photosFor, setPhotosFor] = useState<SchedulePatient | null>(null);
  const [photosUrls, setPhotosUrls] = useState<string[]>([]);
  const [photosLoading, setPhotosLoading] = useState(false);
  // Submit guards (stop double-clicks while in-flight)
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [editSubmitting, setEditSubmitting] = useState(false);
  const [importing, setImporting] = useState(false);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<ScheduleItem | null>(null);
  const [filters, setFilters] = useState<Record<string, string>>({});
  // Bumped to force-remount the filter inputs when resetting to default mode.
  const [filterKey, setFilterKey] = useState(0);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const { message, modal } = App.useApp();
  const { t } = useTranslation();

  // Watch the overall start/end values so the patient list editor re-renders
  // (and re-runs its clamp effect) when the admin changes the schedule range.
  // getFieldValue alone wouldn't trigger a re-render outside Form.Item.
  const createStart = Form.useWatch('startTime', createForm) as { format?: (f: string) => string } | undefined;
  const createEnd = Form.useWatch('endTime', createForm) as { format?: (f: string) => string } | undefined;
  const editStart = Form.useWatch('startTime', editForm) as { format?: (f: string) => string } | undefined;
  const editEnd = Form.useWatch('endTime', editForm) as { format?: (f: string) => string } | undefined;
  const createOverallStart = createStart?.format?.('HH:mm') ?? '09:00';
  const createOverallEnd = createEnd?.format?.('HH:mm') ?? '12:00';
  const editOverallStart = editStart?.format?.('HH:mm') ?? editingRecord?.startTime ?? '09:00';
  const editOverallEnd = editEnd?.format?.('HH:mm') ?? editingRecord?.endTime ?? '12:00';
  // Year of the schedule date drives the per-patient已實付 hint in the editor.
  const createDate = Form.useWatch('date', createForm) as { year?: () => number } | undefined;
  const editDate = Form.useWatch('date', editForm) as { year?: () => number } | undefined;
  const createYear = createDate?.year?.();
  const editYear = editDate?.year?.();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getAdminSchedules(filters);
      setData(list);
    } catch {
      message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [filters, message, t]);

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

  /** Open the photo preview modal for one completed SchedulePatient. */
  const openPhotos = async (sp: SchedulePatient) => {
    setPhotosFor(sp);
    setPhotosLoading(true);
    try {
      const urls = await getSchedulePatientPhotos(sp.id);
      setPhotosUrls(urls);
    } catch (err) {
      message.error(extractApiError(err) || t('common.failed'));
      setPhotosUrls([]);
    } finally {
      setPhotosLoading(false);
    }
  };

  const handleCreate = async (values: Record<string, unknown>) => {
    // Re-entrancy guard — Form.onFinish can re-fire if user double-taps the
    // submit button before the first request settles, which would create two
    // schedules. setCreateSubmitting + Button loading is the primary defence
    // but we also bail here just in case.
    if (createSubmitting) return;

    const overallStart = (values.startTime as { format: (f: string) => string }).format('HH:mm');
    const overallEnd = (values.endTime as { format: (f: string) => string }).format('HH:mm');

    const result = validatePatientTimes(overallStart, overallEnd, createPatients);
    if (!result.ok) {
      message.error(t(`errors.${result.code}`));
      return;
    }
    const validPatients = createPatients.filter((p) => p.patientId && p.startTime && p.endTime);

    setCreateSubmitting(true);
    try {
      const payload: Record<string, unknown> = {
        translatorId: values.translatorId as number,
        date: (values.date as { format: (f: string) => string }).format('YYYY-MM-DD'),
        startTime: overallStart,
        endTime: overallEnd,
        location: values.location as string,
        patients: validPatients,
        note: (values.note as string) || undefined,
      };
      if (values.recurrenceRule) {
        payload.recurrenceRule = values.recurrenceRule as string;
        payload.recurrenceUntil = (values.recurrenceUntil as { format: (f: string) => string }).format('YYYY-MM-DD');
      }
      await createSchedule(payload as unknown as Parameters<typeof createSchedule>[0]);
      message.success(t('common.success'));
      setCreateOpen(false);
      createForm.resetFields();
      setCreatePatients([]);
      void fetchData();
    } catch (err: unknown) {
      message.error(extractApiError(err) || t('common.failed'));
    } finally {
      setCreateSubmitting(false);
    }
  };

  const handleEdit = async (values: Record<string, unknown>) => {
    if (!editingRecord) return;
    if (editSubmitting) return; // re-entrancy guard, see handleCreate

    const overallStart = (values.startTime as { format: (f: string) => string }).format('HH:mm');
    const overallEnd = (values.endTime as { format: (f: string) => string }).format('HH:mm');

    if (editPatients.length > 0) {
      const result = validatePatientTimes(overallStart, overallEnd, editPatients);
      if (!result.ok) {
        message.error(t(`errors.${result.code}`));
        return;
      }
    }
    const validPatients = editPatients.filter((p) => p.patientId && p.startTime && p.endTime);

    setEditSubmitting(true);
    try {
      const payload: Record<string, unknown> = {
        translatorId: values.translatorId as number,
        date: (values.date as { format: (f: string) => string }).format('YYYY-MM-DD'),
        startTime: overallStart,
        endTime: overallEnd,
        location: values.location as string,
        note: (values.note as string) || undefined,
      };
      if (validPatients.length > 0) payload.patients = validPatients;
      await updateSchedule(editingRecord.id, payload);
      message.success(t('common.success'));
      setEditOpen(false);
      void fetchData();
    } catch (err: unknown) {
      message.error(extractApiError(err) || t('common.failed'));
    } finally {
      setEditSubmitting(false);
    }
  };

  const handleDelete = (record: ScheduleItem) => {
    modal.confirm({
      title: t('common.confirm'),
      content: t('schedules.confirmDelete'),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deleteSchedule(record.id);
          message.success(t('common.success'));
          void fetchData();
        } catch {
          message.error(t('common.failed'));
        }
      },
    });
  };

  const handleDeleteGroup = (record: ScheduleItem) => {
    modal.confirm({
      title: t('schedules.deleteGroup'),
      content: t('schedules.confirmDeleteGroup'),
      okText: t('schedules.deleteGroup'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deleteScheduleGroup(record.id);
          message.success(t('common.success'));
          void fetchData();
        } catch {
          message.error(t('common.failed'));
        }
      },
    });
  };

  const openEdit = (record: ScheduleItem) => {
    setEditingRecord(record);
    editForm.setFieldsValue({
      translatorId: record.translatorId,
      location: record.location,
      note: record.note,
    });
    // Preload the patient list editor from the existing schedule_patients rows.
    setEditPatients(
      (record.patients ?? []).map((sp) => ({
        patientId: sp.patientId,
        startTime: sp.startTime,
        endTime: sp.endTime,
        prepaidAmount: sp.prepaidAmount,
      })),
    );
    setEditOpen(true);
  };

  const handleDateRangeChange = (_: unknown, dateStrings: [string, string]) => {
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

  // Default ("latest created") mode = no filter applied. The backend then
  // returns the most recently created schedules (created_at DESC, capped at 100).
  const isDefaultMode = Object.keys(filters).length === 0;

  // Resets all filters back to default mode and force-remounts the filter
  // inputs (via filterKey) so their displayed values clear too.
  const resetToDefault = () => {
    setFilters({});
    setFilterKey((k) => k + 1);
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
    { title: t('schedules.date'), dataIndex: 'date', key: 'date', width: 110 },
    {
      title: t('schedules.startTime'),
      key: 'time',
      width: 120,
      render: (_: unknown, r: ScheduleItem) => `${r.startTime} - ${r.endTime}`,
    },
    { title: t('schedules.translator'), dataIndex: 'translatorName', key: 'translatorName' },
    { title: t('schedules.location'), dataIndex: 'location', key: 'location' },
    {
      title: t('schedules.patients'),
      key: 'patients',
      render: (_: unknown, r: ScheduleItem) => {
        if (r.patients && r.patients.length > 0) {
          return r.patients.map((p: SchedulePatient) => p.patientName).join(', ');
        }
        return r.patientName;
      },
    },
    {
      title: t('common.status'),
      dataIndex: 'checkinStatus',
      key: 'checkinStatus',
      render: (status: string) => (
        <Tag color={statusColorMap[status] ?? 'default'}>{status}</Tag>
      ),
    },
    {
      title: t('common.operation'),
      key: 'action',
      render: (_: unknown, record: ScheduleItem) => {
        const pending = (record.patients ?? []).filter((p) => p.status === 'pending').length;
        return (
          <Space wrap>
            <Button size="small" onClick={() => setDetailRecord(record)}>
              {t('common.detail')}
              {pending > 0 && (
                <Tag color="orange" style={{ marginLeft: 4 }}>
                  {pending}
                </Tag>
              )}
            </Button>
            <Button size="small" onClick={() => openEdit(record)}>{t('common.edit')}</Button>
            <Button size="small" danger onClick={() => handleDelete(record)}>{t('common.delete')}</Button>
            {record.recurrenceGroupId && (
              <Button size="small" danger onClick={() => handleDeleteGroup(record)}>
                {t('schedules.deleteGroup')}
              </Button>
            )}
          </Space>
        );
      },
    },
  ];

  const translatorOptions = translators.map((tr) => ({ value: tr.id, label: tr.name }));

  const scheduleFormFields = (
    <>
      <Form.Item name="translatorId" label={t('schedules.translator')} rules={[{ required: true }]}>
        <Select options={translatorOptions} showSearch optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="date" label={t('schedules.date')} rules={[{ required: true }]}>
        <DatePicker style={{ width: '100%' }} />
      </Form.Item>
      <Space style={{ width: '100%' }} size="middle">
        <Form.Item name="startTime" label={t('schedules.startTime')} rules={[{ required: true }]}>
          <TimePicker format="HH:mm" />
        </Form.Item>
        <Form.Item name="endTime" label={t('schedules.endTime')} rules={[{ required: true }]}>
          <TimePicker format="HH:mm" />
        </Form.Item>
      </Space>
      <Form.Item name="location" label={t('schedules.location')} rules={[{ required: true }]}>
        <Input />
      </Form.Item>
      <Form.Item name="note" label={t('schedules.note')}>
        <Input.TextArea rows={2} />
      </Form.Item>
    </>
  );

  const recurrenceFields = (
    <>
      <Form.Item name="recurrenceRule" label={t('schedules.recurrenceRule')}>
        <Select
          allowClear
          placeholder={t('schedules.recurrenceOptions.none')}
          options={[
            { value: 'daily', label: t('schedules.recurrenceOptions.daily') },
            { value: 'weekly:1,3,5', label: t('schedules.recurrenceOptions.weekly135') },
            { value: 'weekly:2,4', label: t('schedules.recurrenceOptions.weekly24') },
            { value: 'weekly:1,2,3,4,5', label: t('schedules.recurrenceOptions.weekly12345') },
            { value: 'monthly:1', label: t('schedules.recurrenceOptions.monthly1') },
            { value: 'monthly:15', label: t('schedules.recurrenceOptions.monthly15') },
          ]}
        />
      </Form.Item>
      <Form.Item
        noStyle
        shouldUpdate={(prev, curr) => prev.recurrenceRule !== curr.recurrenceRule}
      >
        {({ getFieldValue }) =>
          getFieldValue('recurrenceRule') ? (
            <Form.Item name="recurrenceUntil" label={t('schedules.recurrenceUntil')} rules={[{ required: true }]}>
              <DatePicker style={{ width: '100%' }} />
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
        <Button
          type={isDefaultMode ? 'primary' : 'default'}
          icon={<ClockCircleOutlined />}
          onClick={resetToDefault}
        >
          {t('schedules.latestCreated')}
        </Button>
        <RangePicker key={`range-${filterKey}`} onChange={handleDateRangeChange} />
        <Select
          key={`tr-${filterKey}`}
          style={{ width: 160 }}
          allowClear
          placeholder={t('schedules.filterTranslator')}
          options={translatorOptions}
          onChange={(v) => handleTranslatorFilter(v ? String(v) : '')}
          showSearch
          optionFilterProp="label"
        />
        <Input.Search
          key={`loc-${filterKey}`}
          style={{ width: 200 }}
          placeholder={t('schedules.searchLocation')}
          allowClear
          onSearch={handleLocationSearch}
        />
        <div style={{ flex: 1 }} />
        <Button icon={<DownloadOutlined />} onClick={downloadImportTemplate}>
          {t('schedules.downloadTemplate')}
        </Button>
        <Upload
          {...({
            accept: '.xlsx,.xls',
            showUploadList: false,
            // Ignore the file pick if a previous import is still running.
            // Otherwise rapid double-clicks would fire two POST /import calls
            // and create duplicate schedules.
            beforeUpload: (file: File) => {
              if (importing) return false;
              setImporting(true);
              importSchedules(file)
                .then((res) => {
                  if (res.failed && res.failed.length > 0) {
                    message.warning(
                      `${t('common.success')}: ${res.successSchedules} (${res.successPatients} patients), ${t('common.failed')}: ${res.failed.length}`,
                    );
                  } else {
                    message.success(
                      `${t('common.success')}: ${res.successSchedules} (${res.successPatients} patients)`,
                    );
                  }
                  void fetchData();
                })
                .catch((err) => message.error(extractApiError(err) || t('common.failed')))
                .finally(() => setImporting(false));
              return false;
            },
          } as UploadProps)}
        >
          <Tooltip title={t('schedules.importHint')}>
            <Button icon={<UploadOutlined />} loading={importing} disabled={importing}>{t('schedules.import')}</Button>
          </Tooltip>
        </Upload>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          {t('schedules.add')}
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
        title={t('schedules.add')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        footer={null}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto' } }}
      >
        <Form form={createForm} onFinish={handleCreate} layout="vertical">
          {scheduleFormFields}
          <Form.Item label={t('schedules.patients')}>
            <SchedulePatientListEditor
              value={createPatients}
              onChange={setCreatePatients}
              overallStart={createOverallStart}
              overallEnd={createOverallEnd}
              scheduleYear={createYear}
            />
          </Form.Item>
          {recurrenceFields}
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={createSubmitting} disabled={createSubmitting}>
              {t('common.create')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('schedules.edit')} open={editOpen} onCancel={() => setEditOpen(false)} footer={null}>
        <Form form={editForm} onFinish={handleEdit} layout="vertical">
          {scheduleFormFields}
          <Form.Item label={t('schedules.patients')}>
            <SchedulePatientListEditor
              value={editPatients}
              onChange={setEditPatients}
              overallStart={editOverallStart}
              overallEnd={editOverallEnd}
              scheduleYear={editYear}
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={editSubmitting} disabled={editSubmitting}>
              {t('common.update')}
            </Button>
          </Form.Item>
        </Form>
      </Modal>

      {/* Admin detail modal — shows patient slots with status and surrogate actions */}
      <Modal
        title={t('common.detail')}
        open={!!detailRecord}
        onCancel={() => setDetailRecord(null)}
        footer={null}
        width={680}
      >
        {detailRecord && (
          <div>
            <p>
              <strong>{detailRecord.date}</strong> {detailRecord.startTime}-{detailRecord.endTime} @ {detailRecord.location}
            </p>
            <p>{t('schedules.translator')}：{detailRecord.translatorName}</p>
            <Table
              size="small"
              dataSource={detailRecord.patients ?? []}
              rowKey="id"
              pagination={false}
              columns={[
                {
                  title: t('common.name'),
                  key: 'patient',
                  render: (_: unknown, r: SchedulePatient) => (
                    <div>
                      <div><strong>{r.patientName}</strong></div>
                      <div style={{ color: '#666', fontSize: 12 }}>📞 {r.patientPhone}</div>
                      <div style={{ color: '#666', fontSize: 12 }}>{r.idType.toUpperCase()}: {r.idNumber}</div>
                    </div>
                  ),
                },
                {
                  title: t('schedules.startTime'),
                  key: 'time',
                  width: 100,
                  render: (_: unknown, r: SchedulePatient) => `${r.startTime}-${r.endTime}`,
                },
                {
                  title: `${t('diagnosis.prepaidAmount')}/${t('diagnosis.actualAmount')}`,
                  key: 'amount',
                  width: 120,
                  render: (_: unknown, r: SchedulePatient) => `${r.prepaidAmount} / ${r.actualAmount}`,
                },
                {
                  title: t('common.status'),
                  dataIndex: 'status',
                  key: 'status',
                  width: 110,
                  render: (s: string) => (
                    <Tag color={spStatusColor[s] ?? 'default'}>
                      {t(`diagnosis.status${s.charAt(0).toUpperCase() + s.slice(1).replace('_show', 'NoShow')}`, {
                        defaultValue: s,
                      })}
                    </Tag>
                  ),
                },
                {
                  // Status-dependent content: completed → view photos button,
                  // no_show → reason text (was previously hidden inside the
                  // ambiguous "Operation" column).
                  title: t('diagnosisResults.photosTitle'),
                  key: 'result',
                  width: 200,
                  render: (_: unknown, r: SchedulePatient) => {
                    if (r.status === 'completed') {
                      return (
                        <Button size="small" onClick={() => openPhotos(r)}>
                          {t('diagnosis.viewPhotos', { defaultValue: 'View photos' })}
                        </Button>
                      );
                    }
                    if (r.status === 'no_show') {
                      return (
                        <Typography.Text type="danger" style={{ fontSize: 12 }}>
                          {r.noShowReason}
                        </Typography.Text>
                      );
                    }
                    return <span style={{ color: '#ccc' }}>—</span>;
                  },
                },
                {
                  title: t('common.operation'),
                  key: 'action',
                  width: 180,
                  render: (_: unknown, r: SchedulePatient) => {
                    // Admins are never locked out: a completed slot can still
                    // have its photos managed (add / delete) — translators lose
                    // this after leave, so the admin is the escalation path.
                    if (r.status === 'completed') {
                      return (
                        <Button size="small" onClick={() => setAdminDiagRow(r)}>
                          {t('diagnosis.managePhotos')}
                        </Button>
                      );
                    }
                    return (
                      <Space>
                        <Button size="small" type="primary" onClick={() => setAdminDiagRow(r)}>
                          {t('diagnosis.upload')}
                        </Button>
                        <Button size="small" danger onClick={() => setAdminNoShowFor(r.id)}>
                          {t('diagnosis.noShow')}
                        </Button>
                      </Space>
                    );
                  },
                },
              ]}
            />
          </div>
        )}
      </Modal>

      {/* Diagnosis photo preview modal — clicked from a "completed" row */}
      <Modal
        title={t('diagnosisResults.photosTitle')}
        open={!!photosFor}
        onCancel={() => { setPhotosFor(null); setPhotosUrls([]); }}
        footer={null}
        width={720}
        destroyOnClose
      >
        {photosFor && (
          <div>
            <div style={{ marginBottom: 12, color: '#666' }}>
              {photosFor.patientName} ・ {photosFor.startTime}-{photosFor.endTime}
            </div>
            {photosLoading ? (
              <div>{t('common.loading')}</div>
            ) : photosUrls.length === 0 ? (
              <div style={{ color: '#999' }}>{t('diagnosisResults.noResult')}</div>
            ) : (
              <Image.PreviewGroup>
                <Space wrap>
                  {photosUrls.map((url) => (
                    <Image
                      key={url}
                      src={url}
                      width={180}
                      style={{ borderRadius: 6 }}
                    />
                  ))}
                </Space>
              </Image.PreviewGroup>
            )}
          </div>
        )}
      </Modal>

      {/* Admin surrogate upload + no-show, reusing the same modals but injecting admin* API calls */}
      {adminDiagRow !== null && (
        <DiagnosisUploadModal
          open={adminDiagRow !== null}
          schedulePatientId={adminDiagRow.id}
          upload={adminUploadDiagnosis}
          listPhotos={adminListDiagnosisPhotos}
          deletePhoto={adminDeleteDiagnosisPhoto}
          prepaidAmount={adminDiagRow.prepaidAmount}
          actualAmount={adminDiagRow.actualAmount}
          setActualAmount={adminSetActualAmount}
          onClose={() => setAdminDiagRow(null)}
          onUploaded={() => {
            void fetchData();
            // Refresh the detail modal record if open
            if (detailRecord) {
              getAdminSchedules({}).then((all) => {
                const updated = all.find((s) => s.id === detailRecord.id);
                if (updated) setDetailRecord(updated);
              });
            }
          }}
        />
      )}
      {adminNoShowFor !== null && (
        <NoShowModal
          open={adminNoShowFor !== null}
          schedulePatientId={adminNoShowFor}
          markNoShow={adminMarkNoShow}
          onClose={() => setAdminNoShowFor(null)}
          onDone={() => {
            void fetchData();
            if (detailRecord) {
              getAdminSchedules({}).then((all) => {
                const updated = all.find((s) => s.id === detailRecord.id);
                if (updated) setDetailRecord(updated);
              });
            }
          }}
        />
      )}
    </div>
  );
}
