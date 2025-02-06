import React, { useState, useEffect, useMemo } from 'react';
import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  List,
  Divider,
  Row,
  Col,
  Card,
  Button,
  Form,
  Input,
  Select,
  Space,
  Switch,
  Flex,
  Tag,
  Spin,
  notification,
  Alert,
  Badge,
} from 'antd';
import {
  ClockCircleOutlined,
  CloudServerOutlined,
  CodeOutlined,
  KeyOutlined,
  LoadingOutlined,
  MinusCircleOutlined,
  RedoOutlined,
  SearchOutlined,
  UserOutlined,
} from '@ant-design/icons';
import TextArea from 'antd/es/input/TextArea';
import {
  DeviceSettings,
  SetDeviceSettingsRequest,
  RegisterToLocatorRequest,
  DeregisterFromLocatorRequest,
} from 'api/platform/server/v1/web_pb';
import {
  deregisterFromLocator,
  disableSecureTunnelling,
  enableSecureTunnelling,
  getDeviceSettings,
  registerToLocator,
  setDeviceSettings,
} from 'api/platform/server/v1/web-WebService_connectquery';
import { HelpModal } from '../../components/HelpModal';
import { Locator } from 'api/platform/daemon/v1/wireguard_pb';

const deviceSettingsHelp = [
  {
    title: 'Timezone',
    avatar: <ClockCircleOutlined />,
    description:
      'Choose the timezone where you live. This makes sure the time on your Home Cloud server works its best.',
  },
  {
    title: 'Username/Password',
    avatar: <UserOutlined />,
    description: 'Change the username and password of your administrator user.',
  },
  {
    title: 'Auto update apps/operating system',
    avatar: <RedoOutlined />,
    description:
      'When enabled, your Home Cloud server will routinely check for and install updates. We recommend keeping these on.',
  },
  {
    title: 'Enable SSH',
    avatar: <CodeOutlined />,
    description:
      "When enabled, your Home Cloud server will be available for SSH connections. We only recommend turning this on if you really know what you're doing.",
  },
  {
    title: 'Trusted SSH keys',
    avatar: <KeyOutlined />,
    description:
      'Instead of using your username and password when logging in over SSH, you can adding public SSH keys here (one per line).',
  },
];

const onTheGoSettingsHelp = [
  {
    title: 'Enable On the Go',
    avatar: <CloudServerOutlined />,
    description:
      'Enabling On the Go will configure your Home Cloud server to accept remote connections so you can access it from your mobile devices while away from home. It does this by creating a secure tunnel between your mobile device and the server. You will need to add at least one Locator server as well so your mobile devices can find your Home Cloud server when away from your house.',
  },
  {
    title: 'Add a Locator',
    avatar: <SearchOutlined />,
    description:
      'You can use the default Locator provided by Home Cloud simply by clicking "Add". You can also connect to a different Locator server as well. If you run your own Locator or know of someone who does, just type in the address and click "Add".',
  },
  {
    title: 'Remove a Locator',
    avatar: <MinusCircleOutlined />,
    description:
      'To remove a Locator, simply find it in the list and click "Remove".',
  },
];

export default function SettingsPage() {
  return (
    <Flex justify="center">
      <Space
        direction="vertical"
        size="large"
        style={{ maxWidth: 450, flex: 'auto' }}
      >
        <Card bordered={false}>
          <Flex justify="space-between">
            <strong>Device Settings</strong>
            <HelpModal
              title="Device Settings Help"
              items={deviceSettingsHelp}
            />
          </Flex>
          <Divider />
          <DeviceSettingsForm />
        </Card>
        <Badge.Ribbon color="orange" text="Experimental" placement="start">
          <Card bordered={false}>
            <Flex justify="space-between">
              <strong>On the Go Settings</strong>
              <HelpModal
                title="On the Go Settings Help"
                items={onTheGoSettingsHelp}
              />
            </Flex>
            <Divider />
            <OnTheGoSettingsForm />
          </Card>
        </Badge.Ribbon>
      </Space>
    </Flex>
  );
}

type DeviceSettingsFormFields = {
  timezone: string;
  username: string;
  password?: string;
  autoUpdateApps: boolean;
  autoUpdateOS: boolean;
  enableSSH: boolean;
  trustedSSHKeys?: string;
};

function DeviceSettingsForm() {
  const [api, contextHolder] = notification.useNotification();
  const [saving, setSaving] = useState(false);
  const [enableSsh, setEnableSsh] = useState(false);
  const { data, error, isLoading } = useQuery(getDeviceSettings);
  const useSetDeviceSettings = useMutation(setDeviceSettings, {
    onSuccess(data, variables, context) {
      setSaving(false);
    },
    onError(error, variables, context) {
      setSaving(false);
      api['warning']({
        message: 'Failed to save settings',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });

  var settings = useMemo(() => new DeviceSettings(), []);
  if (data?.settings) {
    settings = data.settings;
  }

  useEffect(() => {
    setEnableSsh(settings.enableSsh);
  }, [settings]);

  const handleSave = (values: DeviceSettingsFormFields) => {
    setSaving(true);
    useSetDeviceSettings.mutate(
      new SetDeviceSettingsRequest({
        settings: {
          adminUser: {
            username: values.username,
            password: values.password,
          },
          timezone: values.timezone,
          autoUpdateApps: values.autoUpdateApps,
          autoUpdateOs: values.autoUpdateOS,
          enableSsh: values.enableSSH,
          trustedSshKeys: values.trustedSSHKeys?.split('\n'),
        },
      })
    );
  };

  const fields: DeviceSettingsFormFields = {
    timezone: settings.timezone,
    username: settings.adminUser?.username || '',
    autoUpdateApps: settings.autoUpdateApps,
    autoUpdateOS: settings.autoUpdateOs,
    enableSSH: settings.enableSsh,
    trustedSSHKeys: settings.trustedSshKeys.join('\n'),
  };

  return (
    <>
      {contextHolder}
      {isLoading && <Spin indicator={<LoadingOutlined spin />} size="large" />}
      {error && (
        <Alert
          message="Failed to load device settings"
          description={error.message}
          type="error"
          showIcon
        />
      )}
      {!isLoading && !error && (
        <Form<DeviceSettingsFormFields>
          name="device-settings"
          layout="vertical"
          initialValues={fields}
          onFinish={handleSave}
          autoComplete="off"
          requiredMark="optional"
          disabled={saving || error != null}
        >
          <Form.Item<DeviceSettingsFormFields>
            label="Timezone"
            name="timezone"
            rules={[{ required: true, message: 'Please select a timezone' }]}
          >
            <Select>
              <Select.Option value="America/New_York">
                Eastern (US)
              </Select.Option>
              <Select.Option value="America/Chicago">
                Central (US)
              </Select.Option>
              <Select.Option value="America/Denver">
                Mountain (US)
              </Select.Option>
              <Select.Option value="America/Los_Angeles">
                Pacific (US)
              </Select.Option>
            </Select>
          </Form.Item>
          <Form.Item<DeviceSettingsFormFields>
            label="Username"
            name="username"
            rules={[{ required: true, message: 'Please select a username' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item<DeviceSettingsFormFields> label="Password" name="password">
            <Input.Password placeholder="(leave blank for no change)" />
          </Form.Item>
          <Form.Item<DeviceSettingsFormFields>
            label="Auto update apps"
            name="autoUpdateApps"
            rules={[{ required: true }]}
          >
            <Switch />
          </Form.Item>
          <Form.Item<DeviceSettingsFormFields>
            label="Auto update system"
            name="autoUpdateOS"
            rules={[{ required: true }]}
          >
            <Switch />
          </Form.Item>
          <Form.Item<DeviceSettingsFormFields>
            label="Enable SSH"
            name="enableSSH"
            rules={[{ required: true }]}
          >
            <Switch onChange={() => setEnableSsh(!enableSsh)} />
          </Form.Item>
          {enableSsh && (
            <Form.Item<DeviceSettingsFormFields>
              label="Trusted SSH keys"
              name="trustedSSHKeys"
            >
              <TextArea placeholder="Enter one key per line" />
            </Form.Item>
          )}
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={saving}>
              Save
            </Button>
          </Form.Item>
        </Form>
      )}
    </>
  );
}

type OnTheGoSettingsFormFields = {
  locatorAddress: string;
};

type DeregisteringFlagMap = {
  [address: string]: boolean;
};

function OnTheGoSettingsForm() {
  const [api, contextHolder] = notification.useNotification();
  const [enableOnTheGo, setEnableOnTheGo] = useState(false);
  const [switching, setSwitching] = useState(false);
  const [registering, setRegistering] = useState(false);
  const [deregistering, setDeregistering] = useState<DeregisteringFlagMap>({});
  const [locators, setLocators] = useState<Locator[]>([]);

  const { data, error, isLoading } = useQuery(getDeviceSettings);
  const useEnableSecureTunnelling = useMutation(enableSecureTunnelling, {
    onSuccess(data, variables, context) {
      setEnableOnTheGo(true);
      setSwitching(false);
    },
    onError(error, variables, context) {
      setSwitching(false);
      api['warning']({
        message: 'Failed to enable On the Go',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const useDisableSecureTunnelling = useMutation(disableSecureTunnelling, {
    onSuccess(data, variables, context) {
      setEnableOnTheGo(false);
      setSwitching(false);
      setLocators([]);
    },
    onError(error, variables, context) {
      setSwitching(false);
      api['warning']({
        message: 'Failed to disable On the Go',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const useRegisterToLocator = useMutation(registerToLocator, {
    onSuccess(data, variables, context) {
      setRegistering(false);
      if (data?.locator) {
        deregistering[data.locator?.address] = false;
        const l = data?.locator;
        setLocators((locators) => [...locators, l]);
      }
    },
    onError(error, variables, context) {
      setRegistering(false);
      api['warning']({
        message: 'Failed to register Locator',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const useDeregisterToLocator = useMutation(deregisterFromLocator, {
    onSuccess(data, variables, context) {
      deregistering[data.locatorAddress] = false;
      setLocators(
        locators.filter(function (l) {
          return l.address !== data.locatorAddress;
        })
      );
    },
    onError(error, variables, context) {
      if (variables.locatorAddress) {
        deregistering[variables.locatorAddress] = false;
      }
      api['warning']({
        message: 'Failed to remove Locator',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });

  var settings = useMemo(() => new DeviceSettings(), []);
  if (data?.settings) {
    settings = data.settings;
  }

  useEffect(() => {
    if (settings.locatorSettings) {
      setEnableOnTheGo(settings.locatorSettings.enabled);
      if (settings.locatorSettings.locators) {
        const tmp = deregistering;
        settings.locatorSettings.locators.forEach((l) => {
          tmp[l.address] = false;
        });
        setDeregistering(tmp);
        setLocators(settings.locatorSettings.locators);
      }
    }
  }, [settings, deregistering]);

  const handleEnableSwitch = (enable: boolean) => {
    setSwitching(true);
    if (enable) {
      useEnableSecureTunnelling.mutate({});
    } else {
      useDisableSecureTunnelling.mutate({});
    }
  };

  const handleRegister = (values: OnTheGoSettingsFormFields) => {
    setRegistering(true);
    useRegisterToLocator.mutate(
      new RegisterToLocatorRequest({
        locatorAddress: values.locatorAddress,
      })
    );
  };

  const handleDeregister = (locatorAddress: string) => {
    deregistering[locatorAddress] = true;
    useDeregisterToLocator.mutate(
      new DeregisterFromLocatorRequest({
        locatorAddress: locatorAddress,
      })
    );
  };

  const fields: OnTheGoSettingsFormFields = {
    locatorAddress: 'https://locator.home-cloud.io',
  };

  return (
    <>
      {contextHolder}
      {isLoading && <Spin indicator={<LoadingOutlined spin />} size="large" />}
      {error && (
        <Alert
          message="Failed to load On the Go settings"
          description={error.message}
          type="error"
          showIcon
        />
      )}
      {!isLoading && !error && (
        <>
          <Row>
            <p>Enable On the Go</p>
          </Row>
          <Row>
            <Switch
              checked={enableOnTheGo}
              loading={switching}
              onChange={() => handleEnableSwitch(!enableOnTheGo)}
            />
          </Row>
        </>
      )}
      {enableOnTheGo && (
        <>
          <Divider />
          <p>Enter Locator server address:</p>
          <Form<OnTheGoSettingsFormFields>
            name="on-the-go-settings"
            layout="inline"
            initialValues={fields}
            onFinish={handleRegister}
            autoComplete="off"
            requiredMark="optional"
            disabled={registering}
          >
            <Button type="primary" htmlType="submit" loading={registering}>
              Add
            </Button>
            <Form.Item<OnTheGoSettingsFormFields>
              name="locatorAddress"
              rules={[
                {
                  required: true,
                  message: '(e.g. https://locator.home-cloud.io)',
                },
              ]}
            >
              <Input style={{ width: '150%' }} />
            </Form.Item>
          </Form>
          <Divider />
          <strong>Active Locators:</strong>
          <List
            itemLayout="horizontal"
            dataSource={locators}
            renderItem={(item) => (
              <List.Item>
                <Card>
                  <Row justify={'space-between'}>
                    <Col flex={1}>
                      <strong>{item.address}</strong>
                    </Col>
                    <Col span={1}>
                      <Button
                        color="danger"
                        variant="outlined"
                        onClick={() => handleDeregister(item.address)}
                        loading={deregistering[item.address]}
                      >
                        Remove
                      </Button>
                    </Col>
                  </Row>
                  <Row>
                    <Flex gap="4px 0" wrap>
                      {item.connections.map((connection, index) => {
                        return (
                          <Tag color="green" key={index}>
                            {connection.wireguardInterface}:
                            {connection.serverId}
                          </Tag>
                        );
                      })}
                    </Flex>
                  </Row>
                </Card>
              </List.Item>
            )}
          ></List>
        </>
      )}
    </>
  );
}
