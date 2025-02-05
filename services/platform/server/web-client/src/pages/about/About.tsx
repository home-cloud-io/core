import React, { useState, useEffect } from 'react';
import {
  Flex,
  Space,
  Card,
  Tag,
  Spin,
  Alert,
  Image,
  List,
  Divider,
  Typography,
  Skeleton,
  Select,
  SelectProps,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useQuery } from '@connectrpc/connect-query';
import { getComponentVersions } from 'api/platform/server/v1/web-WebService_connectquery';
import logo from '../../assets/logo.png';
import {
  LogListener,
  LogProvider,
  ProviderValue,
  useLogs,
} from '../../services/Logs';
import { Log } from 'api/platform/server/v1/web_pb';

export default function AboutPage() {
  return (
    <Flex justify="center">
      <Space direction="vertical" size="large" style={{ flex: 'auto' }}>
        <Details />
        <LogProvider>
          <LogListener />
          <Logs />
        </LogProvider>
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

type DropdownOptions = Map<string, boolean>;

export function Logs() {
  const { log, connected } = useLogs() as ProviderValue;
  const [logs, setLogs] = useState<Log[]>([]);
  const [limit, setLimit] = useState(0);
  const [loading, setLoading] = useState(false);
  const [domains, setDomains] = useState<SelectProps['options']>([
  ]);
  const [sources, setSources] = useState<SelectProps['options']>([
  ]);

  useEffect(() => {
    if (log) {
      logs.unshift(log);
      setLogs(logs);
      setLimit(limit + 1);
      if (domains) {
        let opt = {
          label: log.domain,
          value: log.domain,
        };
        if (domains.indexOf(opt) === -1) {
          domains.push(opt);
          domains.sort();
          setDomains(domains);
        }
      }
      if (sources) {
        let opt = {
          label: log.source,
          value: log.source,
        };
        if (sources.indexOf(opt) === -1) {
          sources.push(opt);
          sources.sort();
          setSources(sources);
        }
      }
    }
  }, [log]);

  const loadMoreData = () => {
    if (loading) {
      return;
    }
    setLoading(true);
    let newLimit = limit + 10;
    if (newLimit > logs.length) {
      newLimit = logs.length;
    }
    setLimit(newLimit);
  };

  return (
    <Card bordered={false}>
      <Flex gap="small" justify="space-between">
        <strong>System Logs</strong>
        {connected && (
          <Tag icon={<CheckCircleOutlined />} color="success">
            Live
          </Tag>
        )}
        {!connected && (
          <Tag icon={<CloseCircleOutlined />} color="error">
            Historical
          </Tag>
        )}
      </Flex>
      <Divider />
      <Flex gap="small" align="start" justify='start'>
        <Select
          mode="multiple"
          allowClear
          placeholder="Domains"
          style={{ width: 120 }}
          options={domains}
        />
        <Select
          mode="multiple"
          allowClear
          placeholder="Components"
          style={{ width: 120 }}
          options={sources}
        />
      </Flex>
      <InfiniteScroll
        dataLength={limit}
        next={loadMoreData}
        hasMore={limit < logs.length}
        loader={<Skeleton paragraph={{ rows: 1 }} active />}
        endMessage={<Divider plain>end of logs</Divider>}
        scrollableTarget="scrollableDiv"
      >
        <List
          size="small"
          dataSource={logs}
          renderItem={(log) => (
            <List.Item>
              <Flex gap="small" justify="space-between">
                <strong>{log.source}</strong>
                <Typography
                  style={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}
                >
                  {log.log}
                </Typography>
              </Flex>
            </List.Item>
          )}
        />
      </InfiniteScroll>
    </Card>
  );
}
