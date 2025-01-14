import * as React from 'react';
import {
  Flex,
  Space,
  Divider,
  Progress,
  Card,
  Row,
  Col,
  Tag,
  Spin,
  Badge,
  Alert,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import { AppHealth, AppStatus } from 'api/platform/server/v1/web_pb';
import { useQuery } from '@connectrpc/connect-query';
import {
  appsHealthCheck,
  getSystemStats,
} from 'api/platform/server/v1/web-WebService_connectquery';
import { SystemStats } from 'api/platform/daemon/v1/system_pb';
import { ProviderValue, useEvents } from '../../services/Subscribe';

export default function HomePage() {
  return (
    <Flex justify="center">
      <Space
        direction="vertical"
        size="large"
        style={{ maxWidth: 450, flex: 'auto' }}
      >
        <DeviceDetails />
        <InstalledApplicationsList />
      </Space>
    </Flex>
  );
}

export function DeviceDetails() {
  const { data, error, isLoading } = useQuery(getSystemStats);
  const { connected } = useEvents() as ProviderValue;

  var stats = new SystemStats();
  if (data?.stats) {
    stats = data.stats;
  }

  const formatBytes = (bytes: number, decimals = 2) => {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  };

  const formatPercentage = (free: number, total: number) => {
    return Math.round(((total - free) / total) * 100);
  };

  return (
    <Card bordered={false}>
      <Row justify="space-between">
        <Col span={4}>
          <strong>Status</strong>
        </Col>
        <Col span={4}>
          {connected && (
            <Tag icon={<CheckCircleOutlined />} color="success">
              Online
            </Tag>
          )}
          {!connected && (
            <Tag icon={<CloseCircleOutlined />} color="error">
              Offline
            </Tag>
          )}
        </Col>
      </Row>
      <Divider />
      <Space direction="vertical" size="small" style={{ display: 'flex' }}>
        <strong>Storage</strong>
        {isLoading && (
          <Spin indicator={<LoadingOutlined spin />} size="large" />
        )}
        {error && (
          <Alert
            message="Failed to load device usage"
            description={error.message}
            type="error"
            showIcon
          />
        )}
        {!isLoading && !error && (
          <div>
            {`${formatBytes(
              Number(stats.drives[0].totalBytes - stats.drives[0].freeBytes)
            )} used out of ${formatBytes(Number(stats.drives[0].totalBytes))} total`}
            <Progress
              percent={formatPercentage(
                Number(stats.drives[0].freeBytes),
                Number(stats.drives[0].totalBytes)
              )}
              percentPosition={{ align: 'start', type: 'outer' }}
            />
          </div>
        )}
      </Space>
    </Card>
  );
}

export function InstalledApplicationsList() {
  const { data, error, isLoading } = useQuery(appsHealthCheck);

  var checks: AppHealth[] = [];
  if (data?.checks) {
    checks = data.checks;
  }

  const ListEntries = () => {
    if (checks) {
      if (checks.length > 0) {
        return (
          <Flex wrap gap="large">
            {checks.map((app) => {
              return <Application key={app.name} app={app} />;
            })}
          </Flex>
        );
      }
    }
    return <p>None</p>;
  };

  return (
    <Card bordered={false}>
      <strong>Applications</strong>
      <Divider />
      {isLoading && <Spin indicator={<LoadingOutlined spin />} size="large" />}
      {error && (
        <Alert
          message="Failed to load applications"
          description={error.message}
          type="error"
          showIcon
        />
      )}
      {!isLoading && !error && <ListEntries />}
    </Card>
  );
}

type Props = {
  app: AppHealth;
};

function Application(props: Props) {
  const app = props.app;

  var status: 'error' | 'success' = 'success';
  if (props.app.status !== AppStatus.HEALTHY) {
    status = 'error';
  }

  return (
    <Col>
      <Space direction="vertical" size="small" style={{ display: 'flex' }}>
        <Row>
          <Badge dot status={status}>
            <img src={app.display?.iconUrl} width={48} height={48} alt="" />
          </Badge>
        </Row>
        <Row>
          <strong className="text-gray-dark">{app.name}</strong>
        </Row>
      </Space>
    </Col>
  );
}
