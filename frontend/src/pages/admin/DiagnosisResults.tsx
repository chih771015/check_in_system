import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Table,
  Tag,
  Typography,
  DatePicker,
  Select,
  Input,
  Space,
  App,
  Button,
  Modal,
  Image,
  Empty,
  Popconfirm,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs, { Dayjs } from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { DiagnosisResult, TranslatorListItem } from '../../types';
import {
  getDiagnosisResults,
  exportDiagnosisResults,
  type DiagnosisResultsQuery,
} from '../../api/diagnosisResults';
import { getTranslators } from '../../api/translators';
import {
  adminUploadDiagnosis,
  adminListDiagnosisPhotos,
  adminDeleteDiagnosisPhoto,
  adminMarkNoShow,
  adminSetActualAmount,
} from '../../api/checkins';
import DiagnosisUploadModal from '../../components/DiagnosisUploadModal';
import NoShowModal from '../../components/NoShowModal';
import { extractApiError } from '../../utils/apiError';

const { RangePicker } = DatePicker;

const statusColor: Record<string, string> = {
  completed: 'green',
  no_show: 'red',
};
const statusKey: Record<string, string> = {
  completed: 'diagnosis.statusCompleted',
  no_show: 'diagnosis.statusNoShow',
};

export default function DiagnosisResultsPage() {
  const { t } = useTranslation();
  const { message } = App.useApp();

  // Default to the last 7 days per Q4 decision.
  const [range, setRange] = useState<[Dayjs, Dayjs]>([dayjs().subtract(6, 'day'), dayjs()]);
  const [translatorId, setTranslatorId] = useState<number | undefined>();
  const [status, setStatus] = useState<'completed' | 'no_show' | undefined>();
  const [patientName, setPatientName] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const [data, setData] = useState<DiagnosisResult[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [translators, setTranslators] = useState<TranslatorListItem[]>([]);
  const [photoOpen, setPhotoOpen] = useState<DiagnosisResult | null>(null);
  // Admin surrogate edit (no leave-lock): manage photos / change status here
  // without going through Schedule Management.
  const [manageFor, setManageFor] = useState<DiagnosisResult | null>(null);
  const [noShowFor, setNoShowFor] = useState<DiagnosisResult | null>(null);

  const query: DiagnosisResultsQuery = useMemo(
    () => ({
      dateFrom: range[0].format('YYYY-MM-DD'),
      dateTo: range[1].format('YYYY-MM-DD'),
      translatorId,
      status,
      patientName: patientName || undefined,
      page,
      pageSize,
    }),
    [range, translatorId, status, patientName, page, pageSize],
  );

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getDiagnosisResults(query);
      setData(res.data);
      setTotal(res.total);
    } catch (err) {
      message.error(extractApiError(err) || t('common.failed'));
    } finally {
      setLoading(false);
    }
  }, [query, message, t]);

  useEffect(() => { void fetchData(); }, [fetchData]);
  useEffect(() => {
    void getTranslators().then(setTranslators).catch(() => undefined);
  }, []);

  const columns: ColumnsType<DiagnosisResult> = [
    {
      title: t('schedules.date'),
      key: 'date',
      width: 170,
      render: (_, r) => (
        <div>
          <div>{r.date}</div>
          <div style={{ color: '#999', fontSize: 12 }}>{r.startTime} - {r.endTime}</div>
        </div>
      ),
    },
    {
      title: t('schedules.translator'),
      dataIndex: 'translatorName',
      key: 'translator',
      width: 130,
    },
    {
      title: t('diagnosisResults.patientInfo'),
      key: 'patient',
      render: (_, r) => (
        <div>
          <div><strong>{r.patientName}</strong></div>
          <div style={{ color: '#666', fontSize: 12 }}>
            📞 {r.patientPhone}
          </div>
          <div style={{ color: '#666', fontSize: 12 }}>
            {r.idType.toUpperCase()}: {r.idNumber}
          </div>
        </div>
      ),
    },
    {
      title: t('schedules.location'),
      dataIndex: 'location',
      key: 'location',
      ellipsis: true,
    },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (s: string) => <Tag color={statusColor[s]}>{t(statusKey[s])}</Tag>,
    },
    {
      title: t('diagnosisResults.photosTitle'),
      key: 'photos',
      width: 200,
      render: (_, r) => {
        if (r.status === 'no_show') {
          return (
            <Typography.Text type="danger" style={{ fontSize: 12 }}>
              {r.noShowReason}
            </Typography.Text>
          );
        }
        if (r.diagnosisPhotos.length === 0) return '-';
        return (
          <Button size="small" onClick={() => setPhotoOpen(r)}>
            {t('diagnosisResults.viewPhotos', { count: r.diagnosisPhotos.length })}
          </Button>
        );
      },
    },
    {
      title: t('diagnosis.prepaidAmount'),
      dataIndex: 'prepaidAmount',
      key: 'prepaidAmount',
      width: 100,
    },
    {
      title: t('diagnosis.actualAmount'),
      dataIndex: 'actualAmount',
      key: 'actualAmount',
      width: 100,
    },
    {
      title: t('diagnosisResults.updated'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: t('common.operation'),
      key: 'action',
      width: 200,
      fixed: 'right',
      render: (_, r) => (
        <Space wrap>
          {/* Admins can manage photos directly here (add / delete). On a no_show
              row, uploading a photo flips the status back to completed. */}
          <Button size="small" onClick={() => setManageFor(r)}>
            {t('diagnosis.managePhotos')}
          </Button>
          {r.status === 'completed' && (
            // Warn first: marking no-show purges this slot's photos.
            <Popconfirm
              title={t('diagnosis.noShowClearsPhotosConfirm')}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
              onConfirm={() => setNoShowFor(r)}
            >
              <Button size="small" danger>
                {t('diagnosis.noShow')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const translatorOptions = translators.map((tr) => ({ value: tr.id, label: tr.name }));

  return (
    <div>
      <Typography.Title level={4}>{t('diagnosisResults.title')}</Typography.Title>

      <Space wrap style={{ marginBottom: 16 }}>
        <RangePicker
          value={range}
          onChange={(v) => {
            if (v && v[0] && v[1]) {
              setRange([v[0], v[1]]);
              setPage(1);
            }
          }}
        />
        <Select
          allowClear
          placeholder={t('diagnosisResults.filterTranslator')}
          style={{ width: 180 }}
          value={translatorId}
          onChange={(v) => { setTranslatorId(v); setPage(1); }}
          options={translatorOptions}
          showSearch
          optionFilterProp="label"
        />
        <Select
          allowClear
          placeholder={t('diagnosisResults.filterStatus')}
          style={{ width: 160 }}
          value={status}
          onChange={(v) => { setStatus(v); setPage(1); }}
          options={[
            { value: 'completed', label: t('diagnosis.statusCompleted') },
            { value: 'no_show', label: t('diagnosis.statusNoShow') },
          ]}
        />
        <Input.Search
          placeholder={t('diagnosisResults.searchPatient')}
          allowClear
          style={{ width: 220 }}
          onSearch={(v) => { setPatientName(v); setPage(1); }}
        />
        <Button onClick={() => void exportDiagnosisResults({ ...query, page: undefined, pageSize: undefined })}>
          {t('checkins.exportExcel')}
        </Button>
      </Space>

      <Table<DiagnosisResult>
        rowKey="schedulePatientId"
        columns={columns}
        dataSource={data}
        loading={loading}
        scroll={{ x: 1300 }}
        locale={{ emptyText: <Empty description={t('diagnosisResults.noResult')} /> }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (p, ps) => { setPage(p); setPageSize(ps); },
        }}
      />

      <Modal
        title={t('diagnosisResults.photosTitle')}
        open={!!photoOpen}
        onCancel={() => setPhotoOpen(null)}
        footer={null}
        width={720}
        destroyOnClose
      >
        {photoOpen && (
          <div>
            <div style={{ marginBottom: 12, color: '#666' }}>
              {photoOpen.patientName} ・ {photoOpen.date} {photoOpen.startTime}-{photoOpen.endTime}
            </div>
            <Image.PreviewGroup>
              <Space wrap>
                {photoOpen.diagnosisPhotos.map((url, i) => (
                  <Image
                    key={i}
                    src={url}
                    width={180}
                    style={{ borderRadius: 6 }}
                  />
                ))}
              </Space>
            </Image.PreviewGroup>
          </div>
        )}
      </Modal>

      {/* Admin surrogate manage modal (photos add/delete) — refetch on change. */}
      {manageFor && (
        <DiagnosisUploadModal
          open={!!manageFor}
          schedulePatientId={manageFor.schedulePatientId}
          upload={adminUploadDiagnosis}
          listPhotos={adminListDiagnosisPhotos}
          deletePhoto={adminDeleteDiagnosisPhoto}
          prepaidAmount={manageFor.prepaidAmount}
          actualAmount={manageFor.actualAmount}
          setActualAmount={adminSetActualAmount}
          onClose={() => setManageFor(null)}
          onUploaded={fetchData}
        />
      )}

      {/* Admin surrogate mark-no-show (purges photos, sets no_show). */}
      {noShowFor && (
        <NoShowModal
          open={!!noShowFor}
          schedulePatientId={noShowFor.schedulePatientId}
          markNoShow={adminMarkNoShow}
          onClose={() => setNoShowFor(null)}
          onDone={fetchData}
        />
      )}
    </div>
  );
}
