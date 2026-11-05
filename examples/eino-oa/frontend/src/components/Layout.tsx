import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Avatar, Dropdown, Layout, Menu, Space, Tag, Typography } from 'antd';
import {
  BookOutlined,
  KeyOutlined,
  LogoutOutlined,
  RobotOutlined,
  TrophyOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useAuth } from '../auth/context';

const { Sider, Content, Header } = Layout;
const { Text } = Typography;

const menuItems = [
  { key: '/students', icon: <UserOutlined />, label: '学生管理' },
  { key: '/courses', icon: <BookOutlined />, label: '课程管理' },
  { key: '/grades', icon: <TrophyOutlined />, label: '成绩管理' },
  { key: '/chat', icon: <RobotOutlined />, label: 'AI 助手' },
  { key: '/tokens', icon: <KeyOutlined />, label: '访问令牌' },
];

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuth();

  const onMenuClick = async ({ key }: { key: string }) => {
    if (key === 'logout') {
      await logout();
    }
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed} theme="dark">
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontSize: collapsed ? 16 : 20,
            fontWeight: 700,
            letterSpacing: 2,
          }}
        >
          {collapsed ? 'OA' : '学生管理系统'}
        </div>
        <Menu
          theme="dark"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          <span style={{ fontSize: 16, fontWeight: 600 }}>大学生信息管理系统</span>
          <Dropdown
            menu={{
              items: [
                { key: 'logout', label: '退出登录', icon: <LogoutOutlined />, danger: true },
              ],
              onClick: onMenuClick,
            }}
            placement="bottomRight"
          >
            <Space style={{ cursor: 'pointer' }}>
              <Avatar size="small" icon={<UserOutlined />} />
              <Text>{user?.display_name || user?.username}</Text>
              {user?.role === 'admin' && <Tag color="blue">管理员</Tag>}
            </Space>
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
