import React, { useState, useEffect, useMemo } from "react";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import {
  List,
  Divider,
  Row,
  Card,
  Button,
  Form,
  Input,
  Select,
  Space,
  Switch,
  Flex,
  Spin,
  notification,
  Alert,
  Badge,
  Modal,
} from "antd";
import {
  ClockCircleOutlined,
  CloudServerOutlined,
  CodeOutlined,
  KeyOutlined,
  LoadingOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  RedoOutlined,
  SearchOutlined,
  UserOutlined,
} from "@ant-design/icons";
import TextArea from "antd/es/input/TextArea";
import {
  DeviceSettings,
  SetDeviceSettingsRequest,
  RegisterToLocatorRequest,
  DeregisterFromLocatorRequest,
} from "api/platform/server/v1/web_pb";
import {
  deregisterFromLocator,
  disableSecureTunnelling,
  enableSecureTunnelling,
  getDeviceSettings,
  registerToLocator,
  setDeviceSettings,
} from "api/platform/server/v1/web-WebService_connectquery";
import { HelpModal } from "../../components/HelpModal";
import { Option } from "antd/es/mentions";

const deviceSettingsHelp = [
  {
    title: "Timezone",
    avatar: <ClockCircleOutlined />,
    description:
      "Choose the timezone where you live. This makes sure the time on your Home Cloud server works its best.",
  },
  {
    title: "Username/Password",
    avatar: <UserOutlined />,
    description: "Change the username and password of your administrator user.",
  },
  {
    title: "Auto update apps/operating system",
    avatar: <RedoOutlined />,
    description:
      "When enabled, your Home Cloud server will routinely check for and install updates. We recommend keeping these on.",
  },
  {
    title: "Enable SSH",
    avatar: <CodeOutlined />,
    description:
      "When enabled, your Home Cloud server will be available for SSH connections. We only recommend turning this on if you really know what you're doing.",
  },
  {
    title: "Trusted SSH keys",
    avatar: <KeyOutlined />,
    description:
      "Instead of using your username and password when logging in over SSH, you can adding public SSH keys here (one per line).",
  },
];

const onTheGoSettingsHelp = [
  {
    title: "Enable On the Go",
    avatar: <CloudServerOutlined />,
    description:
      "Enabling On the Go will configure your Home Cloud server to accept remote connections so you can access it from your mobile devices while away from home. It does this by creating a secure tunnel between your mobile device and the server. You will need to add at least one Locator server as well so your mobile devices can find your Home Cloud server when away from your house.",
  },
  {
    title: "Add a Locator",
    avatar: <SearchOutlined />,
    description:
      'You can use the default Locator provided by Home Cloud simply by clicking "Add". You can also connect to a different Locator server as well. If you run your own Locator or know of someone who does, just type in the address and click "Add".',
  },
  {
    title: "Remove a Locator",
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
        style={{ maxWidth: 450, flex: "auto" }}
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
          <Card>
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
      api["warning"]({
        message: "Failed to save settings",
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: "bottomRight",
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
          trustedSshKeys: values.trustedSSHKeys?.split("\n"),
        },
      })
    );
  };

  const fields: DeviceSettingsFormFields = {
    timezone: settings.timezone,
    username: settings.adminUser?.username || "",
    autoUpdateApps: settings.autoUpdateApps,
    autoUpdateOS: settings.autoUpdateOs,
    enableSSH: settings.enableSsh,
    trustedSSHKeys: settings.trustedSshKeys.join("\n"),
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
            rules={[{ required: true, message: "Please select a timezone" }]}
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
            rules={[{ required: true, message: "Please select a username" }]}
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

function OnTheGoSettingsForm() {
  const [api, contextHolder] = notification.useNotification();
  const [enableOnTheGo, setEnableOnTheGo] = useState(false);
  const [switching, setSwitching] = useState(false);
  const [modalActive, setModalActive] = useState(false);
  const [locators, setLocators] = useState<string[]>([]);

  const { data, error, isLoading, refetch } = useQuery(getDeviceSettings);
  const useEnableSecureTunnelling = useMutation(enableSecureTunnelling, {
    onSuccess(data, variables, context) {
      setEnableOnTheGo(true);
      setSwitching(false);
      refetch();
    },
    onError(error, variables, context) {
      setSwitching(false);
      api["warning"]({
        message: "Failed to enable On the Go",
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: "bottomRight",
      });
    },
  });
  const useDisableSecureTunnelling = useMutation(disableSecureTunnelling, {
    onSuccess(data, variables, context) {
      setEnableOnTheGo(false);
      setSwitching(false);
      refetch();
    },
    onError(error, variables, context) {
      setSwitching(false);
      api["warning"]({
        message: "Failed to disable On the Go",
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: "bottomRight",
      });
    },
  });

  const handleEnableSwitch = (enable: boolean) => {
    setSwitching(true);
    if (enable) {
      useEnableSecureTunnelling.mutate({});
    } else {
      useDisableSecureTunnelling.mutate({});
    }
  };

  useEffect(() => {
    if (data?.settings?.secureTunnelingSettings) {
      setEnableOnTheGo(data.settings.secureTunnelingSettings.enabled);

      if (
        data.settings.secureTunnelingSettings.wireguardInterfaces.length > 0
      ) {
        setLocators(
          data.settings.secureTunnelingSettings?.wireguardInterfaces[0]
            .locatorServers
        );
      }
    }
  }, [data]);

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
          <List
            header={
              <Flex justify="space-between">
                <strong>Locators</strong>
                <Button
                  type="primary"
                  shape="circle"
                  icon={<PlusOutlined />}
                  onClick={() => setModalActive(true)}
                />
              </Flex>
            }
            itemLayout="horizontal"
            dataSource={locators}
            renderItem={(item) => (
              <LocatorListItem address={item} refetch={refetch} />
            )}
          ></List>
          <AddLocatorModal
            active={modalActive}
            setActive={setModalActive}
            refetch={refetch}
          />
        </>
      )}
    </>
  );
}

type LocatorListItemProps = {
  address: string;
  refetch: () => void;
};

function LocatorListItem(props: LocatorListItemProps) {
  const [api, contextHolder] = notification.useNotification();
  const [deregistering, setDeregistering] = useState(false);
  const useDeregisterToLocator = useMutation(deregisterFromLocator, {
    onSuccess(data, variables, context) {
      setDeregistering(false);
      props.refetch()
    },
    onError(error, variables, context) {
      setDeregistering(false);
      api["warning"]({
        message: "Failed to remove Locator",
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: "bottomRight",
      });
    },
  });

  const handleDeregister = (locatorAddress: string) => {
    setDeregistering(true);
    useDeregisterToLocator.mutate(
      new DeregisterFromLocatorRequest({
        locatorAddress: locatorAddress,
        wireguardInterface: "wg0",
      })
    );
  };
  return (
    <>
      {contextHolder}
      <List.Item
        actions={[
          <Button
            color="danger"
            variant="outlined"
            onClick={() => handleDeregister(props.address)}
            loading={deregistering}
          >
            Remove
          </Button>,
        ]}
      >
        <Flex justify="space-between">{props.address}</Flex>
      </List.Item>
    </>
  );
}

type AddLocatorModalProps = {
  active: boolean;
  setActive: React.Dispatch<React.SetStateAction<boolean>>;
  refetch: () => void;
};

type RegisterLocatorFormFields = {
  stockSelection: string;
  customSelection: string;
};

function AddLocatorModal(props: AddLocatorModalProps) {
  const [api, contextHolder] = notification.useNotification();
  const [registering, setRegistering] = useState(false);

  const useRegisterToLocator = useMutation(registerToLocator, {
    onSuccess(data, variables, context) {
      setRegistering(false);
      props.refetch();
      props.setActive(false);
    },
    onError(error, variables, context) {
      setRegistering(false);
      api["warning"]({
        message: "Failed to register Locator",
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: "bottomRight",
      });
    },
  });

  const handleRegister = (values: RegisterLocatorFormFields) => {
    setRegistering(true);
    useRegisterToLocator.mutate(
      new RegisterToLocatorRequest({
        locatorAddress: values.stockSelection === "custom" ? values.customSelection : values.stockSelection,
        wireguardInterface: "wg0",
      })
    );
  };

  const fields: RegisterLocatorFormFields = {
    stockSelection: "",
    customSelection: "",
  };

  return (
    <>
      {contextHolder}
      <Modal open={props.active} closable={false} footer={<></>}>
        <strong>Add Locator</strong>
        <p>
          You can use one of the default Locator servers provided by Home Cloud
          simply by selecting one from the dropdown. You can also connect to a
          custom Locator server by selecting "custom" from the dropdown and
          entering the address of the server.
        </p>
        <Form<RegisterLocatorFormFields>
          name="register-locator"
          initialValues={fields}
          onFinish={handleRegister}
          autoComplete="off"
          requiredMark="optional"
          disabled={registering}
        >
          <Form.Item<RegisterLocatorFormFields>
            name="stockSelection"
            rules={[
              {
                required: true,
                message: "(e.g. https://locator.example.com)",
              },
            ]}
          >
            <Select placeholder="Select a Locator..." allowClear>
              <Option value="https://locator1.home-cloud.io" />
              <Option value="https://locator2.home-cloud.io" />
              <Option value="custom" />
            </Select>
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.stockSelection !== currentValues.stockSelection
            }
          >
            {({ getFieldValue }) =>
              getFieldValue("stockSelection") === "custom" ? (
                <Form.Item
                  name="customSelection"
                  label="Address"
                  rules={[
                    {
                      required: true,
                      message: "(e.g. https://locator.example.com)",
                    },
                  ]}
                >
                  <Input />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Flex justify="space-between">
            <Button variant="outlined" onClick={() => props.setActive(false)}>
              Cancel
            </Button>
            <Button type="primary" htmlType="submit" loading={registering}>
              Add
            </Button>
          </Flex>
        </Form>
      </Modal>
    </>
  );
}
