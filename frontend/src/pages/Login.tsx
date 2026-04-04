import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Card, Typography, App } from 'antd';
import { MailOutlined, LockOutlined } from '@ant-design/icons';
import { login as loginApi } from '../api/auth';
import { useAuth } from '../stores/authStore';

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const { message } = App.useApp();

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
    } catch {
      message.error('登入失敗，請檢查帳號密碼');
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
          翻譯員打卡系統
        </Typography.Title>
        <Form onFinish={onFinish} layout="vertical">
          <Form.Item name="email" rules={[{ required: true, message: '請輸入電子信箱' }]}>
            <Input prefix={<MailOutlined />} placeholder="電子信箱" size="large" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '請輸入密碼' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密碼" size="large" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block size="large" loading={loading}>
              登入
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
