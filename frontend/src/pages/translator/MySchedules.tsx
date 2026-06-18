import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, Button, Tag, Switch, Typography, Spin, Empty, App, Space, Tooltip } from 'antd';
import {
  EnvironmentOutlined,
  ClockCircleOutlined,
  UserOutlined,
  CheckCircleFilled,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ScheduleItem, SchedulePatient } from '../../types';
import { getMySchedules } from '../../api/schedules';
import DiagnosisUploadModal from '../../components/DiagnosisUploadModal';
import NoShowModal from '../../components/NoShowModal';

const statusColorMap: Record<string, string> = {
  none: 'default',
  arrived: 'orange',
  completed: 'green',
  makeup: 'blue',
};

const statusLabelKey: Record<string, string> = {
  none: 'checkins.type.arrive',
  arrived: 'checkins.type.arrive',
  completed: 'common.success',
  makeup: 'checkins.isMakeup',
};

const spStatusColor: Record<string, string> = {
  pending: 'orange',
  completed: 'green',
  no_show: 'red',
};
const spStatusKey: Record<string, string> = {
  pending: 'diagnosis.statusPending',
  completed: 'diagnosis.statusCompleted',
  no_show: 'diagnosis.statusNoShow',
};

export default function MySchedules() {
  const [schedules, setSchedules] = useState<ScheduleItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  // Modal state — null means closed; non-null is the SchedulePatient.id.
  // diagCanUpload / diagCanDelete gate the modal's affordances per context.
  const [diagFor, setDiagFor] = useState<number | null>(null);
  const [diagCanUpload, setDiagCanUpload] = useState(true);
  const [diagCanDelete, setDiagCanDelete] = useState(true);
  const [diagPrepaid, setDiagPrepaid] = useState(0);
  const [diagActual, setDiagActual] = useState(0);
  const [noShowFor, setNoShowFor] = useState<number | null>(null);

  const openDiag = (p: SchedulePatient, canUpload: boolean, canDelete: boolean) => {
    setDiagCanUpload(canUpload);
    setDiagCanDelete(canDelete);
    setDiagPrepaid(p.prepaidAmount);
    setDiagActual(p.actualAmount);
    setDiagFor(p.id);
  };
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const fetchSchedules = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (!showHistory) params.date_from = new Date().toISOString().slice(0, 10);
      const list = await getMySchedules(params);
      setSchedules(list);
    } catch {
      message.error(t('errors.INTERNAL_ERROR'));
    } finally {
      setLoading(false);
    }
  }, [showHistory, message, t]);

  useEffect(() => {
    void fetchSchedules();
  }, [fetchSchedules]);

  const isPast = (schedule: ScheduleItem) => {
    const now = new Date();
    const today = now.toISOString().slice(0, 10);
    if (schedule.date < today) return true;
    if (schedule.date > today) return false;
    const [endH, endM] = schedule.endTime.split(':').map(Number);
    const endAt = new Date();
    endAt.setHours(endH, endM, 0, 0);
    return now > endAt;
  };

  /** Number of patients still in pending status for this schedule. */
  const pendingCount = (s: ScheduleItem) =>
    (s.patients ?? []).filter((p) => p.status === 'pending').length;

  const renderActions = (schedule: ScheduleItem) => {
    const past = isPast(schedule);

    if (schedule.checkinStatus === 'completed') {
      return <Tag icon={<CheckCircleFilled />} color="green">{t('common.success')}</Tag>;
    }
    if (schedule.checkinStatus === 'none' && !past) {
      return (
        <Button type="primary" onClick={() => navigate(`/checkin/${schedule.id}/arrive`)}>
          {t('checkin.checkinType.arrive')}
        </Button>
      );
    }
    if (schedule.checkinStatus === 'arrived' && !past) {
      // Stage 4: gate the leave button until all patients are processed.
      const pending = pendingCount(schedule);
      const button = (
        <Button
          type="primary"
          disabled={pending > 0}
          onClick={() => navigate(`/checkin/${schedule.id}/leave`)}
        >
          {t('checkin.checkinType.leave')}
        </Button>
      );
      return pending > 0 ? (
        <Tooltip title={t('diagnosis.pendingPatients', { count: pending })}>
          <span>{button}</span>
        </Tooltip>
      ) : button;
    }
    if (schedule.checkinStatus === 'none' && past) {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/arrive`)}>
          {t('checkins.isMakeup')} ({t('checkins.type.arrive')})
        </Button>
      );
    }
    if (schedule.checkinStatus === 'arrived' && past) {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/leave`)}>
          {t('checkins.isMakeup')} ({t('checkins.type.leave')})
        </Button>
      );
    }
    if (schedule.checkinStatus === 'makeup') {
      return (
        <Button onClick={() => navigate(`/makeup/${schedule.id}/leave`)}>
          {t('checkins.isMakeup')} ({t('checkins.type.leave')})
        </Button>
      );
    }
    return null;
  };

  /** Renders one patient row inside a schedule card. */
  const renderPatient = (s: ScheduleItem, p: SchedulePatient) => {
    // Three contexts:
    //  - in progress (arrived/makeup): full edit (add + delete + no-show)
    //  - left (schedule completed): append-only — late X-ray/lab results can be
    //    added, but delete / no-show stay locked (backend also enforces this)
    //  - before arrive (none): view-only
    const inProgress = s.checkinStatus === 'arrived' || s.checkinStatus === 'makeup';
    const left = s.checkinStatus === 'completed';
    return (
      <div
        key={p.id}
        style={{
          marginLeft: 18,
          fontSize: 13,
          padding: '8px 0',
          borderBottom: '1px dashed #eee',
        }}
      >
        <Space size="small" wrap style={{ width: '100%' }}>
          <Typography.Text strong>{p.patientName}</Typography.Text>
          <Tag color={spStatusColor[p.status]}>{t(spStatusKey[p.status])}</Tag>
        </Space>
        <div style={{ color: '#666', marginTop: 2 }}>
          📞 {p.patientPhone} ・ {p.idType.toUpperCase()}: {p.idNumber}
        </div>
        <div style={{ color: '#999', fontSize: 12, marginTop: 2 }}>
          {p.startTime} - {p.endTime}
        </div>
        <div style={{ color: '#666', fontSize: 12, marginTop: 2 }}>
          {t('diagnosis.prepaidAmount')}: {p.prepaidAmount} ・ {t('diagnosis.actualAmount')}: {p.actualAmount}
        </div>
        {p.status === 'no_show' && p.noShowReason && (
          <div style={{ color: '#cf1322', fontSize: 12, marginTop: 2 }}>
            — {p.noShowReason}
          </div>
        )}
        <Space size="small" style={{ marginTop: 6 }} wrap>
          {inProgress ? (
            <>
              <Button size="small" type="primary" onClick={() => openDiag(p, true, true)}>
                {p.status === 'completed' ? t('diagnosis.managePhotos') : t('diagnosis.upload')}
              </Button>
              {p.status !== 'completed' && (
                <Button size="small" danger onClick={() => setNoShowFor(p.id)}>
                  {t('diagnosis.noShow')}
                </Button>
              )}
            </>
          ) : left ? (
            // Append-only after leave (add late results; no delete / no-show).
            // Offered for completed (supplement) and pending (late result), not no_show.
            p.status !== 'no_show' && (
              <Button size="small" type="primary" onClick={() => openDiag(p, true, false)}>
                {t('diagnosis.supplementPhotos')}
              </Button>
            )
          ) : (
            // Before arrive: view-only, only when there is something to see.
            p.status === 'completed' && (
              <Button size="small" onClick={() => openDiag(p, false, false)}>
                {t('diagnosis.viewPhotos')}
              </Button>
            )
          )}
        </Space>
      </div>
    );
  };

  return (
    <div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Typography.Title level={4} style={{ margin: 0 }}>
          {t('nav.mySchedules')}
        </Typography.Title>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Typography.Text>{t('schedules.showHistory')}</Typography.Text>
          <Switch checked={showHistory} onChange={setShowHistory} />
        </div>
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : schedules.length === 0 ? (
        <Empty />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {schedules.map((s) => (
            <Card key={s.id} size="small">
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'flex-start',
                  flexWrap: 'wrap',
                  gap: 8,
                }}
              >
                <div style={{ flex: 1, minWidth: 200 }}>
                  <div style={{ marginBottom: 4 }}>
                    <Typography.Text strong>{s.date}</Typography.Text>
                    <Tag color={statusColorMap[s.checkinStatus] ?? 'default'} style={{ marginLeft: 8 }}>
                      {t(statusLabelKey[s.checkinStatus] ?? 'common.status')}
                    </Tag>
                  </div>
                  <div style={{ color: '#666', fontSize: 14 }}>
                    <div>
                      <ClockCircleOutlined style={{ marginRight: 4 }} />
                      {s.startTime} - {s.endTime}
                    </div>
                    <div>
                      <EnvironmentOutlined style={{ marginRight: 4 }} />
                      {s.location}
                    </div>
                    {s.patients && s.patients.length > 0 ? (
                      <div>
                        <UserOutlined style={{ marginRight: 4 }} />
                        {s.patients.map((p) => renderPatient(s, p))}
                      </div>
                    ) : (
                      <div>
                        <UserOutlined style={{ marginRight: 4 }} />
                        {s.patientName}
                      </div>
                    )}
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  {renderActions(s)}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {diagFor !== null && (
        <DiagnosisUploadModal
          open={diagFor !== null}
          schedulePatientId={diagFor}
          canUpload={diagCanUpload}
          canDelete={diagCanDelete}
          prepaidAmount={diagPrepaid}
          actualAmount={diagActual}
          onClose={() => setDiagFor(null)}
          onUploaded={fetchSchedules}
        />
      )}
      {noShowFor !== null && (
        <NoShowModal
          open={noShowFor !== null}
          schedulePatientId={noShowFor}
          onClose={() => setNoShowFor(null)}
          onDone={fetchSchedules}
        />
      )}
    </div>
  );
}
