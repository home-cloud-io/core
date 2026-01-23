import React, { useState } from 'react';
import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  deleteApp,
  getAppsInStore,
  installApp,
} from '@home-cloud/api/platform/server/v1/web-WebService_connectquery';
import {
  App,
  DeleteAppRequest,
  InstallAppRequest,
} from '@home-cloud/api/platform/server/v1/web_pb';
import {
  Alert,
  Spin,
  Card,
  Divider,
  List,
  Button,
  Avatar,
  Modal,
  Flex,
  notification,
  Tag,
} from 'antd';
import {
  AppstoreAddOutlined,
  LoadingOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import { marked } from 'marked';
import { HelpModal } from '../../components/HelpModal';
import { ProviderValue, useEvents } from '../../services/Subscribe';

const help = [
  {
    title: 'Installing Apps',
    avatar: <AppstoreAddOutlined />,
    description:
      'Here you can install apps to your Home Cloud server. Simply click "More Info", then "Install" on any app you want to use.',
  },
  {
    title: 'Uninstalling Apps',
    avatar: <MinusCircleOutlined />,
    description:
      'You can also remove any app you currently have installed and no longer need. Simply click "More Info", then "Uninstall".',
  },
];

const statusMap: Record<string, string | undefined> = {
  alpha: 'red',
  beta: 'orange',
  stable: 'green',
};

export default function AppStorePage() {
  const [api, contextHolder] = notification.useNotification();
  const { event } = useEvents() as ProviderValue;
  const { data, error, isLoading } = useQuery(getAppsInStore);
  const useInstallApp = useMutation(installApp, {
    onSuccess(data, variables, context) {
      // TODO
    },
    onError(error, variables, context) {
      api['warning']({
        message: 'Failed to install App',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const useDeleteApp = useMutation(deleteApp, {
    onSuccess(data, variables, context) {
      // TODO
    },
    onError(error, variables, context) {
      api['warning']({
        message: 'Failed to uninstall App',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });

  var apps: App[] = [];
  if (data?.apps) {
    apps = data.apps;
  }

  if (event?.event.case === 'appInstalled') {
    const e = event.event.value;
    apps.forEach((app) => {
      if (app.name === e.name) {
        app.installed = true;
      }
    });
  }

  const handleInstall = (app: App) => {
    useInstallApp.mutate(
      new InstallAppRequest({
        repo: 'apps.home-cloud.io',
        chart: app.name,
        release: app.name,
        version: app.version,
      })
    );
  };

  const handleUninstall = (app: App) => {
    // TODO: handle this with an event loop from the server
    app.installed = false;
    useDeleteApp.mutate(
      new DeleteAppRequest({
        release: app.name,
      })
    );
  };

  return (
    <>
      {contextHolder}
      <Flex justify="center">
        <Card bordered={false} style={{ maxWidth: 650, flex: 'auto' }}>
          <Flex justify={'space-between'}>
            <strong>App Store</strong>
            <HelpModal title="App Store Help" items={help} />
          </Flex>
          <Divider />
          {isLoading && (
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          )}
          {error && (
            <Alert
              message="Failed to load Apps"
              description={error.message}
              type="error"
              showIcon
            />
          )}
          {!isLoading && !error && (
            <List
              itemLayout="vertical"
              dataSource={apps}
              renderItem={(app) => (
                <AppItem
                  key={app.name}
                  app={app}
                  handleInstall={handleInstall}
                  handleUninstall={handleUninstall}
                />
              )}
            ></List>
          )}
        </Card>
      </Flex>
    </>
  );
}

type AppItemProps = {
  app: App;
  handleInstall: any;
  handleUninstall: any;
};

function AppItem(props: AppItemProps) {
  const [active, setActive] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [uninstalling, setUninstalling] = useState(false);
  const app = props.app;

  if (app.installed && installing) {
    setInstalling(false);
  }

  if (!app.installed && uninstalling) {
    setUninstalling(false);
  }

  const showModal = () => {
    setActive(true);
  };

  const handleCancel = () => {
    setActive(false);
  };

  const handleInstallClick = () => {
    setInstalling(true);
    props.handleInstall(app);
  };

  const handleUninstallClick = () => {
    setUninstalling(true);
    props.handleUninstall(app);
  };

  return (
    <div>
      <List.Item
        key={app.digest}
        actions={[
          <p>
            App: {app.appVersion} | Release: {app.version}
          </p>,
        ]}
        extra={<Button onClick={() => showModal()}>More Info</Button>}
      >
        <List.Item.Meta
          avatar={<Avatar src={app.icon} />}
          title={app.annotations['displayName']}
          description={app.description}
        ></List.Item.Meta>
      </List.Item>
      <Modal
        open={active}
        onCancel={() => handleCancel()}
        footer={() => (
          <>
            {app.installed && (
              <Button
                color="danger"
                variant="outlined"
                onClick={handleUninstallClick}
                loading={uninstalling}
              >
                Uninstall
              </Button>
            )}
            {!app.installed && (
              <Button
                color="primary"
                variant="solid"
                onClick={handleInstallClick}
                loading={installing}
              >
                Install
              </Button>
            )}
          </>
        )}
      >
        <Flex gap="middle" vertical={false}>
          <img src={app.icon} alt="" />
          <Flex gap="small" vertical justify="center" align="center">
            <Tag color="grey">{app.appVersion}</Tag>
            <Tag color={statusMap[app.annotations['status']]}>
              {app.annotations['status']}
            </Tag>
          </Flex>
          <Flex gap="small" vertical justify="center" align="center">
            <Button onClick={() => window.open(app.home, '_blank')}>
              Website
            </Button>
          </Flex>
        </Flex>
        <div
          dangerouslySetInnerHTML={{
            __html: marked.parse(app.readme).toString(),
          }}
        />
      </Modal>
      <Divider />
    </div>
  );
}
