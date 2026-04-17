import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Card, Typography, App } from 'antd';
import { LockOutlined } from '@ant-design/icons';
import { changePassword } from '../api/auth';
import { useAuth } from '../stores/authStore';

export default function ChangePasswordPage() {
  const [loading, setLoading] = useState(false);
  const { user, login, isAdmin } = useAuth();
  const navigate = useNavigate();
  const { message } = App.useApp();

  const onFinish = async (values: {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
  }) => {
    if (values.newPassword !== values.confirmPassword) {
      message.error('新密碼與確認密碼不一致');
      return;
    }
    if (!user) {
      message.error('登入狀態異常，請重新登入');
      return;
    }
    setLoading(true);
    try {
      const resp = await changePassword(values.oldPassword, values.newPassword);
      // Replace the stale token (which still carries must_change_pw=true)
      // with the new one returned from the backend.
      login(resp.token, { ...user, mustChangePW: false });
      message.success('密碼已更新');
      if (isAdmin) {
        navigate('/admin/translators');
      } else {
        navigate('/my-schedules');
      }
    } catch {
      message.error('密碼更新失敗，請確認舊密碼是否正確');
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
          變更密碼
        </Typography.Title>
        {user?.mustChangePW && (
          <Typography.Paragraph type="warning" style={{ textAlign: 'center' }}>
            首次登入請先變更密碼
          </Typography.Paragraph>
        )}
        <Form onFinish={onFinish} layout="vertical">
          <Form.Item
            name="oldPassword"
            label="舊密碼"
            rules={[{ required: true, message: '請輸入舊密碼' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="舊密碼" size="large" />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label="新密碼"
            rules={[
              { required: true, message: '請輸入新密碼' },
              { min: 6, message: '密碼至少6個字元' },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="新密碼" size="large" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="確認新密碼"
            rules={[{ required: true, message: '請再次輸入新密碼' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="確認新密碼" size="large" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              確認變更
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
