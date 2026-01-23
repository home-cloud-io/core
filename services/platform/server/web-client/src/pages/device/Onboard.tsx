import {
  Alert,
  Button,
  Card,
  Carousel,
  Divider,
  Flex,
  Form,
  Input,
  notification,
  Select,
  Space,
  Spin,
  Switch,
} from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import React, { useEffect, useRef, useState } from 'react';
import { CarouselRef } from 'antd/es/carousel';
import { useMutation, useQuery } from '@connectrpc/connect-query';
import {
  initializeDevice,
  isDeviceSetup,
} from '@home-cloud/api/platform/server/v1/web-WebService_connectquery';
import { InitializeDeviceRequest } from '@home-cloud/api/platform/server/v1/web_pb';
import { useNavigate } from 'react-router-dom';

const logo = require('../../assets/logo.png');

type DeviceOnboardProps = {
  setDisabled: React.Dispatch<React.SetStateAction<boolean>>;
};

export default function DeviceOnboardPage(props: DeviceOnboardProps) {
  const [api, contextHolder] = notification.useNotification();
  const navigate = useNavigate();
  const { data, error, isLoading } = useQuery(isDeviceSetup);
  let carouselRef = useRef<CarouselRef>(null);
  const useInitializeDevice = useMutation(initializeDevice, {
    onMutate(variables) {
      setSubmitting(true);
    },
    onSuccess(data, variables, context) {
      setSubmitting(false);
      props.setDisabled(false);
      navigate('/');
    },
    onError(error, variables, context) {
      setSubmitting(false);
      api['warning']({
        message: 'Failed to initialize device',
        description: error.rawMessage,
        showProgress: true,
        pauseOnHover: true,
        placement: 'bottomRight',
      });
    },
  });
  const [submitting, setSubmitting] = useState(false);
  const [request, setRequest] = useState(
    new InitializeDeviceRequest({
      username: '',
      password: '',
      timezone: '',
      autoUpdateApps: true,
      autoUpdateOs: true,
    })
  );

  if (data && data.setup) {
    props.setDisabled(false);
    navigate('/');
  }

  const handleSubmit = () => {
    useInitializeDevice.mutate(request);
  };

  return (
    <>
      {contextHolder}
      <Flex justify="center">
        {isLoading && (
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        )}
        {error && (
          <Alert
            message="Failed to connect to device"
            description={error.message}
            type="error"
            showIcon
          />
        )}
        {!isLoading && !error && (
          <Carousel infinite={false} style={{ width: 450 }} ref={carouselRef}>
            <WelcomeCard
              next={carouselRef.current?.next}
              prev={carouselRef.current?.prev}
              request={request}
              setRequest={setRequest}
              submit={handleSubmit}
              submitting={submitting}
            />
            <UserSetupCard
              next={carouselRef.current?.next}
              prev={carouselRef.current?.prev}
              request={request}
              setRequest={setRequest}
              submit={handleSubmit}
              submitting={submitting}
            />
            <DeviceSettingsCard
              next={carouselRef.current?.next}
              prev={carouselRef.current?.prev}
              request={request}
              setRequest={setRequest}
              submit={handleSubmit}
              submitting={submitting}
            />
          </Carousel>
        )}
      </Flex>
    </>
  );
}

type Props = {
  next: (() => void) | undefined;
  prev: (() => void) | undefined;
  request: InitializeDeviceRequest;
  setRequest: React.Dispatch<React.SetStateAction<InitializeDeviceRequest>>;
  submit: () => void;
  submitting: boolean;
};

function WelcomeCard(props: Props) {
  return (
    <Card bordered={false}>
      <Flex justify="center">
        <img src={logo} width={106} height={58} alt="the Home Cloud logo which is a purple cloud with the silhouette of a house embedded in it" />
      </Flex>
      <Flex justify="center">
        <h3>Welcome to Home Cloud!</h3>
      </Flex>
      <Space align="center" direction="vertical" style={{ width: '100%' }}>
        <span>Let's walk you through some simple steps to get started.</span>
        <span>You'll be up and running in just a couple of minutes!</span>
      </Space>

      <Divider />

      <Flex justify="flex-end">
        <Button
          type="primary"
          onClick={() => {
            if (props.next) {
              props.next();
            }
          }}
        >
          Next
        </Button>
      </Flex>
    </Card>
  );
}

function UserSetupCard(props: Props) {
  const [isUsernameValid, setUsernameValidity] = useState<'' | 'error'>('');
  const [isPasswordValid, setPasswordValidity] = useState<'' | 'error'>('');

  useEffect(() => {
    if (props.request.username.length >= 4) {
      setUsernameValidity('');
    }

    if (props.request.password.length >= 4) {
      setPasswordValidity('');
    }
  }, [props.request.username, props.request.password]);

  const handleUsernameChange = (username: string) => {
    props.request.username = username;
    props.setRequest(props.request);

    if (username.length < 4) {
      setUsernameValidity('error');
    } else {
      setUsernameValidity('');
    }
  };

  const handlePasswordChange = (password: string) => {
    props.request.password = password;
    props.setRequest(props.request);

    if (password.length < 4) {
      setPasswordValidity('error');
    } else {
      setPasswordValidity('');
    }
  };

  return (
    <Card title="User Setup" bordered={false}>
      <Space direction="vertical" size="large">
        <span>
          Setup the default administrative user. Don't worry you can always
          change it later.
        </span>
        <Form layout="vertical">
          <Form.Item label="Username">
            <Input
              status={isUsernameValid}
              onChange={(e) => handleUsernameChange(e.target.value)}
            />
          </Form.Item>
          <Form.Item label="Password">
            <Input.Password
              status={isPasswordValid}
              onChange={(e) => handlePasswordChange(e.target.value)}
            />
          </Form.Item>
        </Form>
      </Space>
      <Divider />
      <Flex justify="space-between">
        <Button
          type="default"
          onClick={() => {
            if (props.prev) {
              props.prev();
            }
          }}
        >
          Back
        </Button>
        <Button
          type="primary"
          disabled={isPasswordValid === 'error' || isUsernameValid === 'error'}
          onClick={() => {
            if (props.next) {
              props.next();
            }
          }}
        >
          Next
        </Button>
      </Flex>
    </Card>
  );
}

type DeviceSettingsFormFields = {
  timezone: string;
  autoUpdateApps: boolean;
  autoUpdateOS: boolean;
};

function DeviceSettingsCard(props: Props) {
  const handleSave = (values: DeviceSettingsFormFields) => {
    props.request.timezone = values.timezone;
    props.request.autoUpdateApps = values.autoUpdateApps;
    props.request.autoUpdateOs = values.autoUpdateOS;
    props.setRequest(props.request);
    props.submit();
  };

  const fields: DeviceSettingsFormFields = {
    timezone: props.request.timezone,
    autoUpdateApps: props.request.autoUpdateApps,
    autoUpdateOS: props.request.autoUpdateOs,
  };

  return (
    <Card title="Device Settings" bordered={false}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <span>Configure the basic settings of your new device.</span>
        <Form<DeviceSettingsFormFields>
          name="initialize-device-settings"
          layout="vertical"
          initialValues={fields}
          onFinish={handleSave}
          autoComplete="off"
          requiredMark="optional"
          disabled={props.submitting}
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
            label="Auto update apps"
            name="autoUpdateApps"
            rules={[{ required: true }]}
          >
            <Switch disabled />
          </Form.Item>
          <Form.Item<DeviceSettingsFormFields>
            label="Auto update system"
            name="autoUpdateOS"
            rules={[{ required: true }]}
          >
            <Switch disabled />
          </Form.Item>
          <Divider />
          <Flex justify="space-between">
            <Button
              type="default"
              onClick={() => {
                if (props.prev) {
                  props.prev();
                }
              }}
            >
              Back
            </Button>
            <Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                loading={props.submitting}
              >
                Save
              </Button>
            </Form.Item>
          </Flex>
        </Form>
      </Space>
    </Card>
  );
}
