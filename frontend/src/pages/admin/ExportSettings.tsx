import { useEffect, useState } from 'react';
import { Form, Select, InputNumber, Input, Switch, Button, Card, Typography, Alert, App } from 'antd';
import { getExportSchedule, upsertExportSchedule, runExportNow } from '../../api/export';

export default function ExportSettings() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [running, setRunning] = useState(false);
  const [lastRunAt, setLastRunAt] = useState<string | null>(null);
  const { message } = App.useApp();

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
      message.success('設定已儲存');
    } catch {
      message.error('儲存失敗');
    } finally {
      setSaving(false);
    }
  };

  const handleRunNow = async () => {
    setRunning(true);
    try {
      const res = await runExportNow();
      const r = res.result ?? res;
      message.success(
        `匯出完成（${r.rangeFrom} ~ ${r.rangeTo}），已寄至 ${r.emailTo}`,
      );
      setLastRunAt(r.ranAt ?? new Date().toISOString());
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data
          ?.error ?? '執行失敗';
      message.error(msg);
    } finally {
      setRunning(false);
    }
  };

  return (
    <div style={{ maxWidth: 520 }}>
      <Typography.Title level={5} style={{ marginBottom: 16 }}>
        定期匯出設定
      </Typography.Title>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 20 }}
        message="每月到達指定日期時，系統會自動在後台產生報表。若選擇 Google Sheet 格式，需先在伺服器端設定 GOOGLE_CREDENTIALS_FILE 環境變數。"
      />

      <Card loading={loading}>
        <Form form={form} layout="vertical" onFinish={handleSave} initialValues={{ frequency: 'monthly', dayOfMonth: 1, format: 'excel', enabled: true }}>
          <Form.Item name="frequency" label="頻率" rules={[{ required: true }]}>
            <Select options={[{ value: 'monthly', label: '每月' }]} />
          </Form.Item>

          <Form.Item name="dayOfMonth" label="每月幾號執行" rules={[{ required: true }]}>
            <InputNumber min={1} max={28} style={{ width: '100%' }} addonAfter="日" />
          </Form.Item>

          <Form.Item name="format" label="匯出格式" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'excel', label: 'Excel (.xlsx)' },
                { value: 'google_sheet', label: 'Google Sheet' },
              ]}
            />
          </Form.Item>

          <Form.Item name="emailTo" label="通知 Email（選填）">
            <Input placeholder="report@example.com" />
          </Form.Item>

          <Form.Item name="enabled" label="啟用" valuePropName="checked">
            <Switch />
          </Form.Item>

          {lastRunAt && (
            <div style={{ marginBottom: 16, color: '#888', fontSize: 13 }}>
              上次執行：{new Date(lastRunAt).toLocaleString('zh-TW')}
            </div>
          )}

          <Button type="primary" htmlType="submit" loading={saving} block>
            儲存設定
          </Button>
        </Form>

        <Button
          style={{ marginTop: 16 }}
          block
          loading={running}
          onClick={handleRunNow}
        >
          立即執行一次
        </Button>
      </Card>
    </div>
  );
}
