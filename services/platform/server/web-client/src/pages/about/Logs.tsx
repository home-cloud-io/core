import React, { useState, useEffect } from 'react';
import {
  Flex,
  Space,
  Card,
  Tag,
  Alert,
  Divider,
  Typography,
  Table,
  TableColumnsType,
  Button,
} from 'antd';
import { ArrowLeftOutlined, SyncOutlined } from '@ant-design/icons';
import { useQuery } from '@connectrpc/connect-query';
import { getSystemLogs } from 'api/platform/server/v1/web-WebService_connectquery';
import { useNavigate } from 'react-router-dom';

export default function LogsPage() {
  const navigate = useNavigate();
  return (
    <Flex justify="center">
      <Space direction="vertical" size="large" style={{ flex: 'auto' }}>
        <Button onClick={() => navigate('/about')}>
          <ArrowLeftOutlined />
          Back
        </Button>
        <Logs />
      </Space>
    </Flex>
  );
}

interface LogType {
  key: React.Key;
  source: string;
  namespace: string;
  domain: string;
  log: string;
  timestamp: Date | undefined;
}

export function Logs() {
  const { data, error, isLoading, isFetching, refetch } = useQuery(
    getSystemLogs,
    {
      sinceSeconds: 300,
    }
  );
  const [logs, setLogs] = useState<LogType[]>([]);
  const [columns, setColumns] = useState<TableColumnsType<LogType>>([]);

  useEffect(() => {
    if (data?.logs) {
      let a: LogType[] = data.logs.map((log, index) => ({
        key: `${index}`,
        source: log.source,
        namespace: log.namespace,
        domain: log.domain,
        log: log.log,
        timestamp: log.timestamp?.toDate(),
      }));
      setLogs(a);
    }

    if (data) {
      const c: TableColumnsType<LogType> = [
        {
          title: 'Timestamp',
          dataIndex: 'timestamp',
          sortOrder: 'descend',
          sorter: (a, b) => {
            if (a.timestamp && b.timestamp) {
              return a.timestamp.valueOf() - b.timestamp.valueOf();
            }
            return 0;
          },
          onFilter: (value, log) => log.domain.includes(value as string),
          render: (value, record, index) => (
            <>{record.timestamp?.toLocaleString()}</>
          ),
        },
        {
          title: 'Domain',
          dataIndex: 'domain',
          filters: data.domains.map((x) => ({
            text: x,
            value: x,
          })),
          filterMode: 'tree',
          filterSearch: true,
          onFilter: (value, log) => log.domain.includes(value as string),
          render: (value, record, index) => (
            <Tag color={stringToColour(record.domain)}>{record.domain}</Tag>
          ),
        },
        {
          title: 'Group',
          dataIndex: 'namespace',
          filters: data.namespaces.map((x) => ({
            text: x,
            value: x,
          })),
          filterMode: 'tree',
          filterSearch: true,
          onFilter: (value, log) => log.namespace.includes(value as string),
          render: (value, record, index) => (
            <Tag color={stringToColour(record.namespace)}>
              {record.namespace}
            </Tag>
          ),
        },
        {
          title: 'Source',
          dataIndex: 'source',
          filters: data.sources.map((x) => ({
            text: x,
            value: x,
          })),
          filterMode: 'tree',
          filterSearch: true,
          onFilter: (value, log) => log.source.includes(value as string),
          render: (value, record, index) => (
            <Tag color={stringToColour(record.source)}>{record.source}</Tag>
          ),
        },
        {
          title: 'Log',
          dataIndex: 'log',
          render: (value, record, index) => (
            <Typography
              style={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}
            >
              {record.log}
            </Typography>
          ),
        },
      ];
      setColumns(c);
    }
  }, [data]);

  return (
    <Card bordered={false}>
      <Flex gap="small" justify="space-between">
        <strong>System Logs</strong>
        <Button
          onClick={() => {
            refetch();
          }}
        >
          <SyncOutlined spin={isFetching} />
        </Button>
      </Flex>
      <Divider />
      {error && (
        <Alert
          message="Failed to load device information"
          description={error.message}
          type="error"
          showIcon
        />
      )}
      {!error && (
        <>
          <Table<LogType>
            size="small"
            loading={isLoading}
            columns={columns}
            dataSource={logs}
          />
        </>
      )}
    </Card>
  );
}

/* eslint-disable no-bitwise */
const stringToColour = (str: string) => {
  let hash = 0;
  str.split('').forEach((char) => {
    hash = char.charCodeAt(0) + ((hash << 5) - hash);
  });
  let colour = '#';
  for (let i = 0; i < 3; i++) {
    const value = (hash >> (i * 8)) & 0xff;
    colour += value.toString(16).padStart(2, '0');
  }
  return colour;
};
/* eslint-enable no-bitwise */
