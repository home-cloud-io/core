import React from 'react';
import {
  Flex,
  Space,
  Card,
  Tag,
  Spin,
  Alert,
  Image,
  List,
  Button,
} from 'antd';
import {
  FileSearchOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import { useQuery } from '@connectrpc/connect-query';
import {
  getComponentVersions,
} from '@home-cloud/api/platform/server/v1/web-WebService_connectquery';
import logo from '../../assets/logo.png';
import { useNavigate } from 'react-router-dom';

export default function AboutPage() {
  return (
    <Flex justify="center">
      <Space
        direction="vertical"
        size="large"
        style={{ maxWidth: 450, flex: 'auto' }}
      >
        <Details />
        <Links />
      </Space>
    </Flex>
  );
}

export function Details() {
  const { data, error, isLoading } = useQuery(getComponentVersions);

  return (
    <Card title="About" bordered={false}>
      <Flex gap="small" vertical justify="center" align="center">
        <Image src={logo} width="50%" preview={false}></Image>
        <h2>Home Cloud</h2>
      </Flex>

      {isLoading && <Spin indicator={<LoadingOutlined spin />} size="large" />}
      {error && (
        <Alert
          message="Failed to load device information"
          description={error.message}
          type="error"
          showIcon
        />
      )}
      {!isLoading && !error && (
        <>
          <List
            header={<p>Platform Components</p>}
            dataSource={data?.platform}
            renderItem={(component) => (
              <List.Item>
                <strong>{component.name}</strong>
                <Tag color="purple">{component.version}</Tag>
              </List.Item>
            )}
          ></List>
          <List
            header={<p>System Components</p>}
            dataSource={data?.system}
            renderItem={(component) => (
              <List.Item>
                <strong>{component.name}</strong>
                <Tag color="purple">{component.version}</Tag>
              </List.Item>
            )}
          ></List>
        </>
      )}
    </Card>
  );
}

export function Links() {
  const navigate = useNavigate();
  return (
    <Card title="Info" bordered={false}>
      <Flex vertical gap="small" style={{ width: '100%' }}>
        <Button block color="primary" onClick={() => navigate('/about/logs')}>
          <FileSearchOutlined /> Logs
        </Button>
      </Flex>
    </Card>
  );
}
