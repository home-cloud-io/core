import React, { useEffect, useState } from 'react';
import {
  UploadOutlined,
  HomeOutlined,
  SettingOutlined,
  PoweroffOutlined,
  RedoOutlined,
  AppstoreOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import {
  Button,
  Layout,
  Menu,
  // theme,
  Dropdown,
  MenuProps,
  message,
  Flex,
} from 'antd';
import { ConfigProvider } from 'antd';
import { Routes, Route, useNavigate } from 'react-router-dom';

import HomePage from './pages/home/Home';
import UploadPage from './pages/upload/Upload';
import SettingsPage from './pages/settings/Settings';
import AppStorePage from './pages/store/Store';
import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  isDeviceSetup,
  restartHost,
  shutdownHost,
} from '@home-cloud/api/platform/server/v1/web-WebService_connectquery';
import DeviceOnboardPage from './pages/device/Onboard';
import AboutPage from './pages/about/About';

import logo from './assets/logo-white-flat.png';
import UpdatesPage from './pages/about/Updates';
import LogsPage from './pages/about/Logs';
const { Header, Sider, Content } = Layout;

const App: React.FC = () => {
  const [api, contextHolder] = message.useMessage();
  const [collapsed, setCollapsed] = useState(false);
  const [disabled, setDisabled] = useState(false);
  const [primary] = React.useState('#643f91');
  const { data, error } = useQuery(isDeviceSetup);
  const navigate = useNavigate();

  if (error) {
    console.warn(`failed to get device setup: ${error.rawMessage}`);
  }

  useEffect(() => {
    if (data && !data.setup) {
      console.log('redirecting to device setup');
      setCollapsed(true);
      setDisabled(true);
      navigate('/getting-started');
    }
  }, [navigate, data]);

  const useRestartHost = useMutation(restartHost, {
    onSuccess(data, variables, context) {
      api['success']('Restarting...');
    },
    onError(error, variables, context) {
      api['warning'](`Failed to restart: ${error.rawMessage}`);
    },
  });
  const useShutdownHost = useMutation(shutdownHost, {
    onSuccess(data, variables, context) {
      api['success']('Shutting down...');
    },
    onError(error, variables, context) {
      api['warning'](`Failed to shutdown: ${error.rawMessage}`);
    },
  });

  const items: MenuProps['items'] = [
    {
      label: 'Restart',
      key: '1',
      icon: <RedoOutlined />,
      onClick: () => useRestartHost.mutate({}),
    },
    {
      label: 'Shutdown',
      key: '2',
      icon: <PoweroffOutlined />,
      onClick: () => useShutdownHost.mutate({}),
    },
  ];

  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: primary,
        },
        components: {
          Layout: {
            headerBg: primary,
            colorBgBase: '#f0f2f5',
            // colorBgBase: "#141414",
            algorithm: true,
          },
          Carousel: {
            colorBgContainer: primary,
          },
        },
        // TODO: figure this out
        // algorithm: theme.darkAlgorithm
      }}
    >
      <Layout style={{ minHeight: '100vh' }}>
        {contextHolder}
        <Header>
          <Flex justify="space-between" align="center">
            <img
              src={logo}
              height={64}
              alt="the Home Cloud logo which is a white cloud with the silhouette of a house embedded in it"
            />
            <Dropdown menu={{ items }} placement="bottomRight">
              <Button>
                <PoweroffOutlined />
              </Button>
            </Dropdown>
          </Flex>
        </Header>
        <Layout>
          <Sider
            theme="light"
            breakpoint="sm"
            collapsible
            collapsedWidth="32"
            collapsed={collapsed}
            // trigger={null}
            onCollapse={(value) => setCollapsed(value)}
          >
            <Menu
              theme="light"
              mode="inline"
              onSelect={({ key }) => {
                navigate(key);
              }}
              items={[
                {
                  label: 'Home',
                  key: '/',
                  icon: <HomeOutlined />,
                  disabled: disabled,
                },
                {
                  label: 'App Store',
                  key: '/store',
                  icon: <AppstoreOutlined />,
                  disabled: disabled,
                },
                {
                  label: 'Upload',
                  key: '/upload',
                  icon: <UploadOutlined />,
                  disabled: disabled,
                },
                {
                  label: 'Settings',
                  key: '/settings',
                  icon: <SettingOutlined />,
                  disabled: disabled,
                },
                {
                  label: 'About',
                  key: '/about',
                  icon: <InfoCircleOutlined />,
                  disabled: disabled,
                },
              ]}
            />
          </Sider>
          <Content
            style={{
              margin: '16px 16px',
            }}
          >
            <Routes>
              <Route path="/" Component={HomePage} />
              <Route path="/store" Component={AppStorePage} />
              <Route path="/upload" Component={UploadPage} />
              <Route path="/updates" Component={UpdatesPage} />
              <Route path="/settings" Component={SettingsPage} />
              <Route path="/about" Component={AboutPage} />
              <Route path="/about/logs" Component={LogsPage} />
              <Route path="/about/updates" Component={UpdatesPage} />
              <Route
                path="/getting-started"
                element={<DeviceOnboardPage setDisabled={setDisabled} />}
              />
            </Routes>
          </Content>
        </Layout>
      </Layout>
    </ConfigProvider>
  );
};

export default App;
