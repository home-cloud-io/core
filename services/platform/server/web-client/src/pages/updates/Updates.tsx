import React, { useState, useEffect } from 'react';
import {
  Flex,
  Space,
  Card,
  Tag,
  List,
  notification,
  Button,
  Divider,
  Empty,
  Modal,
  Typography,
} from 'antd';
import {
  ArrowRightOutlined,
  CheckCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useMutation } from '@connectrpc/connect-query';
import {
  changeDaemonVersion,
  checkForContainerUpdates,
  checkForSystemUpdates,
  installOSUpdate,
  setSystemImage,
} from 'api/platform/server/v1/web-WebService_connectquery';
import {
  CheckForContainerUpdatesResponse,
  CheckForSystemUpdatesResponse,
  DaemonVersion,
  ImageVersion,
} from 'api/platform/server/v1/web_pb';

export default function UpdatesPage() {
  return (
    <Flex justify="center">
      <Space
        direction="vertical"
        size="large"
        style={{ maxWidth: 450, flex: 'auto' }}
      >
        <Details />
      </Space>
    </Flex>
  );
}

export function Details() {
  return (
    <Card title="Updates" bordered={false}>
      <PlatformComponents />
      <Divider />
      <SystemComponents />
    </Card>
  );
}

function PlatformComponents() {
  const [api, contextHolder] = notification.useNotification();
  const [containerUpdates, setContainerUpdates] =
    useState<CheckForContainerUpdatesResponse>();
  const [loading, setLoading] = useState(false);
  const useCheckForContainerUpdates = useMutation(checkForContainerUpdates, {
    onSuccess(data, variables, context) {
      setLoading(false);
      setContainerUpdates(data);
    },
    onError(error, variables, context) {
      setLoading(false);
      api['warning']({
        message: 'Failed to get platform component updates',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const useSetSystemImage = useMutation(setSystemImage, {
    onSuccess(data, variables, context) {
      handleCheckForUpdates();
    },
    onError(error, variables, context) {
      handleCheckForUpdates();
      api['warning']({
        message: 'Failed to update platform component',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });

  const handleCheckForUpdates = () => {
    setLoading(true);
    useCheckForContainerUpdates.mutate({});
  };

  const handleUpdateImage = (image: ImageVersion) => {
    setLoading(true);
    useSetSystemImage.mutate({
      currentImage: `${image.image}:${image.current}`,
      requestedImage: `${image.image}:${image.latest}`,
    });
  };

  useEffect(() => {
    handleCheckForUpdates();
  }, [])

  return (
    <>
      {contextHolder}
      <List
        header={
          <Flex gap="small" justify="space-between">
            Platform Components
            <Button
              size="small"
              shape="round"
              disabled={loading}
              onClick={() => handleCheckForUpdates()}
            >
              <SyncOutlined spin={loading} />
            </Button>
          </Flex>
        }
        dataSource={containerUpdates?.imageVersions}
        renderItem={(component) => (
          <List.Item>
            <strong>{component.name}</strong>
            {component.current === component.latest ? (
              <Flex gap="none">
                <Tag color="purple">{component.current}</Tag>
                <CheckCircleOutlined style={{ color: 'green' }} />
              </Flex>
            ) : (
              <Flex gap="small">
                <Tag color="purple">{component.current}</Tag>
                <ArrowRightOutlined />
                <Tag color="orange">{component.latest}</Tag>
                <Button
                  size="small"
                  onClick={() => handleUpdateImage(component)}
                >
                  Update
                </Button>
              </Flex>
            )}
          </List.Item>
        )}
      />
    </>
  );
}

function SystemComponents() {
  const [api, contextHolder] = notification.useNotification();
  const [systemUpdates, setSystemUpdates] =
    useState<CheckForSystemUpdatesResponse>();
  const [loading, setLoading] = useState(false);
  const [details, setDetails] = useState(false);
  const useCheckForSystemUpdates = useMutation(checkForSystemUpdates, {
    onSuccess(data, variables, context) {
      setLoading(false);
      setSystemUpdates(data);
    },
    onError(error, variables, context) {
      setLoading(false);
      api['warning']({
        message: 'Failed to get system component updates',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const useChangeDaemonVersion = useMutation(changeDaemonVersion, {
    onSuccess(data, variables, context) {
      handleCheckForUpdates();
    },
    onError(error, variables, context) {
      handleCheckForUpdates();
      api['warning']({
        message: 'Failed to update daemon',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const useInstallOSUpdate = useMutation(installOSUpdate, {
    onSuccess(data, variables, context) {
      handleCheckForUpdates();
    },
    onError(error, variables, context) {
      handleCheckForUpdates();
      api['warning']({
        message: 'Failed to update NixOS',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });

  const showModal = () => {
    setDetails(true);
  };

  const handleCancel = () => {
    setDetails(false);
  };

  const handleCheckForUpdates = () => {
    setLoading(true);
    useCheckForSystemUpdates.mutate({});
  };

  const handleUpdateDaemon = (version: DaemonVersion | undefined) => {
    setLoading(true);
    useChangeDaemonVersion.mutate({
      version: version?.version,
      srcHash: version?.srcHash,
      vendorHash: version?.vendorHash,
    });
  };

  const handleUpdateOS = () => {
    setLoading(true);
    useInstallOSUpdate.mutate({});
  };

  useEffect(() => {
    handleCheckForUpdates();
  }, [])

  return (
    <>
      {contextHolder}
      <>
        <List
          header={
            <Flex gap="small" justify="space-between">
              System Components
              <Button
                size="small"
                shape="round"
                disabled={loading}
                onClick={() => handleCheckForUpdates()}
              >
                <SyncOutlined spin={loading} />
              </Button>
            </Flex>
          }
        >
          {!systemUpdates ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />
          ) : (
            <>
              <List.Item>
                <strong>daemon</strong>
                {systemUpdates?.daemonVersions?.current?.version ===
                systemUpdates?.daemonVersions?.latest?.version ? (
                  <Flex gap="none">
                    <Tag color="purple">
                      {systemUpdates?.daemonVersions?.current?.version}
                    </Tag>
                    <CheckCircleOutlined style={{ color: 'green' }} />
                  </Flex>
                ) : (
                  <Flex gap="small">
                    <Tag color="purple">
                      {systemUpdates?.daemonVersions?.current?.version}
                    </Tag>
                    <ArrowRightOutlined />
                    <Tag color="orange">
                      {systemUpdates?.daemonVersions?.latest?.version}
                    </Tag>
                    <Button
                      size="small"
                      onClick={() =>
                        handleUpdateDaemon(
                          systemUpdates?.daemonVersions?.latest
                        )
                      }
                    >
                      Update
                    </Button>
                  </Flex>
                )}
              </List.Item>
              <List.Item>
                <strong>nixos</strong>
                {systemUpdates?.osDiff.includes('No version or selection state changes.') ? (
                  <Flex gap="none">
                    <Tag color="purple">Latest</Tag>
                    <CheckCircleOutlined style={{ color: 'green' }} />
                  </Flex>
                ) : (
                  <Flex gap="small">
                    <Tag color="purple">current</Tag>
                    <ArrowRightOutlined />
                    <Button
                      variant="outlined"
                      // TODO: figure out how to make this orange like the Tags
                      color="danger"
                      size="small"
                      onClick={() => showModal()}
                    >
                      details
                    </Button>
                    <Button size="small" onClick={() => handleUpdateOS()}>
                      Update
                    </Button>
                  </Flex>
                )}
              </List.Item>
            </>
          )}
        </List>
        <Modal
          title="NixOS Update Details"
          open={details}
          onCancel={() => handleCancel()}
          footer={null}
        >
          <Divider />
          <Typography
            style={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}
          >
            {systemUpdates?.osDiff}
          </Typography>
        </Modal>
      </>
    </>
  );
}
