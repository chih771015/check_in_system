import { useEffect, useState } from 'react';
import { Form, Select, InputNumber, Input, Switch, Button, Card, Typography, Alert, App } from 'antd';
import { useTranslation } from 'react-i18next';
import { getExportSchedule, upsertExportSchedule, runExportNow } from '../../api/export';

export default function ExportSettings() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [running, setRunning] = useState(false);
  const [lastRunAt, setLastRunAt] = useState<string | null>(null);
  const { message } = App.useApp();
  const { t } = useTranslation();

  useEffect(() => {
    setLoading(true);
    getExportSchedule()
      .then((data) => {
        if (data) {
          form.setFieldsValue({
            frequency: data.frequency,
            dayOfMonth: data.dayOfMonth,
            format: data.format,
            emailTo: data.emailTo,
            enabled: data.enabled,
          });
          setLastRunAt(data.lastRunAt ?? null);
        }
      })
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, [form]);

  const handleSave = async (values: Record<string, unknown>) => {
    setSaving(true);
    try {
      await upsertExportSchedule({
        frequency: values.frequency as string,
        dayOfMonth: values.dayOfMonth as number,
        format: values.format as string,
        emailTo: (values.emailTo as string) || '',
        enabled: values.enabled as boolean,
      });
      message.success(t('exportSettings.saved'));
    } catch {
      message.error(t('common.failed'));
    } finally {
      setSaving(false);
    }
  };

  const handleRunNow = async () => {
    setRunning(true);
    try {
      const res = await runExportNow();
      const r = res.result ?? res;
      message.success(t('exportSettings.runSuccess'));
      setLastRunAt(r.ranAt ?? new Date().toISOString());
      void r;
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('common.failed');
      message.error(msg);
    } finally {
      setRunning(false);
    }
  };

  return (
    <div style={{ maxWidth: 520 }}>
      <Typography.Title level={5} style={{ marginBottom: 16 }}>
        {t('exportSettings.title')}
      </Typography.Title>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 20 }}
        message={t('exportSettings.hint')}
      />

      <Card loading={loading}>
        <Form form={form} layout="vertical" onFinish={handleSave} initialValues={{ frequency: 'monthly', dayOfMonth: 1, format: 'excel', enabled: true }}>
          <Form.Item name="frequency" label={t('exportSettings.frequency')} rules={[{ required: true }]}>
            <Select options={[{ value: 'monthly', label: t('exportSettings.frequency') }]} />
          </Form.Item>

          <Form.Item name="dayOfMonth" label={t('exportSettings.dayOfMonth')} rules={[{ required: true }]}>
            <InputNumber min={1} max={28} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="format" label={t('exportSettings.format')} rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'excel', label: 'Excel (.xlsx)' },
                { value: 'google_sheet', label: 'Google Sheet' },
              ]}
            />
          </Form.Item>

          <Form.Item name="emailTo" label={t('exportSettings.emailTo')}>
            <Input placeholder="report@example.com" />
          </Form.Item>

          <Form.Item name="enabled" label={t('exportSettings.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>

          {lastRunAt && (
            <div style={{ marginBottom: 16, color: '#888', fontSize: 13 }}>
              {new Date(lastRunAt).toLocaleString()}
            </div>
          )}

          <Button type="primary" htmlType="submit" loading={saving} block>
            {t('exportSettings.save')}
          </Button>
        </Form>

        <Button
          style={{ marginTop: 16 }}
          block
          loading={running}
          onClick={handleRunNow}
        >
          {t('exportSettings.runNow')}
        </Button>
      </Card>
    </div>
  );
}
