import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout, Menu, Button, Typography, theme } from 'antd';
import {
  TeamOutlined,
  ScheduleOutlined,
  CalendarOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  CheckSquareOutlined,
  SettingOutlined,
  FileSearchOutlined,
  HistoryOutlined,
} from '@ant-design/icons';
import { useAuth } from '../stores/authStore';

const { Header, Sider, Content } = Layout;

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const { user, logout, isAdmin } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { token: themeToken } = theme.useToken();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const adminMenuItems = [
    {
      key: '/admin/translators',
      icon: <TeamOutlined />,
      label: '翻譯員管理',
    },
    {
      key: '/admin/schedules',
      icon: <ScheduleOutlined />,
      label: '排班管理',
    },
    {
      key: '/admin/checkins',
      icon: <CheckSquareOutlined />,
      label: '打卡紀錄',
    },
    {
      key: '/admin/export-settings',
      icon: <SettingOutlined />,
      label: '定期匯出設定',
    },
    {
      key: '/admin/audit-logs',
      icon: <FileSearchOutlined />,
      label: '操作紀錄',
    },
  ];

  const translatorMenuItems = [
    {
      key: '/my-schedules',
      icon: <CalendarOutlined />,
      label: '我的排班',
    },
    {
      key: '/my-checkins',
      icon: <HistoryOutlined />,
      label: '我的打卡紀錄',
    },
  ];

  const menuItems = isAdmin ? adminMenuItems : translatorMenuItems;

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        breakpoint="lg"
        collapsedWidth={0}
        trigger={null}
        style={{ background: themeToken.colorBgContainer }}
      >
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            borderBottom: `1px solid ${themeToken.colorBorderSecondary}`,
          }}
        >
          {!collapsed && (
            <Typography.Text strong style={{ fontSize: 14 }}>
              翻譯員打卡系統
            </Typography.Text>
          )}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 16px',
            background: themeToken.colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: `1px solid ${themeToken.colorBorderSecondary}`,
          }}
        >
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <Typography.Text>{user?.name}</Typography.Text>
            <Button type="text" icon={<LogoutOutlined />} onClick={handleLogout}>
              登出
            </Button>
          </div>
        </Header>
        <Content style={{ margin: 16, padding: 24, background: themeToken.colorBgContainer, borderRadius: 8, overflow: 'auto' }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
