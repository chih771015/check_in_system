import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Card, Typography, App } from 'antd';
import { LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { changePassword } from '../api/auth';
import { useAuth } from '../stores/authStore';

export default function ChangePasswordPage() {
  const [loading, setLoading] = useState(false);
  const { user, login, isAdmin } = useAuth();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const onFinish = async (values: {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
  }) => {
    if (values.newPassword !== values.confirmPassword) {
      message.error(t('changePassword.mismatch'));
      return;
    }
    if (!user) {
      message.error(t('errors.USER_NOT_FOUND'));
      return;
    }
    setLoading(true);
    try {
      const resp = await changePassword(values.oldPassword, values.newPassword);
      login(resp.token, { ...user, mustChangePW: false });
      message.success(t('changePassword.successUpdated'));
      if (isAdmin) {
        navigate('/admin/translators');
      } else {
        navigate('/my-schedules');
      }
    } catch {
      message.error(t('errors.OLD_PASSWORD_INCORRECT'));
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
          {t('changePassword.title')}
        </Typography.Title>
        {user?.mustChangePW && (
          <Typography.Paragraph type="warning" style={{ textAlign: 'center' }}>
            {t('changePassword.mustChange')}
          </Typography.Paragraph>
        )}
        <Form onFinish={onFinish} layout="vertical">
          <Form.Item name="oldPassword" label={t('common.oldPassword')} rules={[{ required: true }]}>
            <Input.Password prefix={<LockOutlined />} placeholder={t('common.oldPassword')} size="large" />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label={t('common.newPassword')}
            rules={[
              { required: true },
              { min: 8, message: t('changePassword.minLength') },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder={t('common.newPassword')} size="large" />
          </Form.Item>
          <Form.Item name="confirmPassword" label={t('common.confirmPassword')} rules={[{ required: true }]}>
            <Input.Password prefix={<LockOutlined />} placeholder={t('common.confirmPassword')} size="large" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              {t('changePassword.submit')}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
