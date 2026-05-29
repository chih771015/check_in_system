import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Card, Typography, App } from 'antd';
import { MailOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { login as loginApi } from '../api/auth';
import { useAuth } from '../stores/authStore';

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const onFinish = async (values: { email: string; password: string }) => {
    setLoading(true);
    try {
      const data = await loginApi(values.email, values.password);
      login(data.token, data.user);
      if (data.user.mustChangePW) {
        navigate('/change-password');
      } else if (data.user.role === 'admin') {
        navigate('/admin/translators');
      } else {
        navigate('/my-schedules');
      }
    } catch (err: unknown) {
      // axios interceptor 已把錯誤碼翻譯放在 translatedMessage / response.data.message
      const translated =
        (err as { translatedMessage?: string })?.translatedMessage ??
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message;
      message.error(translated || t('errors.INVALID_CREDENTIALS'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f5f5f5',
      }}
    >
      <Card style={{ width: 400, maxWidth: '90vw' }}>
        <Typography.Title level={3} style={{ textAlign: 'center', marginBottom: 24 }}>
          {t('app.title')}
        </Typography.Title>
        <Form onFinish={onFinish} layout="vertical">
          <Form.Item name="email" rules={[{ required: true }]}>
            <Input prefix={<MailOutlined />} placeholder={t('login.emailPlaceholder')} size="large" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true }]}>
            <Input.Password prefix={<LockOutlined />} placeholder={t('login.passwordPlaceholder')} size="large" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              {t('login.submit')}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
