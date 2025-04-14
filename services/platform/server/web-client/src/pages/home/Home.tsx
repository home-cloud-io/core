import * as React from "react";
import {
  Flex,
  Space,
  Divider,
  Progress,
  Card,
  Tag,
  Spin,
  Badge,
  Alert,
  Avatar,
  Empty,
  Button,
  Tooltip,
  ProgressProps,
} from "antd";
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from "@ant-design/icons";
import { AppHealth, AppStatus } from "api/platform/server/v1/web_pb";
import { useQuery } from "@connectrpc/connect-query";
import {
  appsHealthCheck,
  getSystemStats,
} from "api/platform/server/v1/web-WebService_connectquery";
import { SystemStats } from "api/platform/daemon/v1/system_pb";
import { ProviderValue, useEvents } from "../../services/Subscribe";
import { useNavigate } from "react-router-dom";

export default function HomePage() {
  return (
    <Flex justify="center">
      <Space
        direction="vertical"
        size="large"
        style={{ maxWidth: 450, flex: "auto" }}
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

  return (
    <Card bordered={false}>
      <Flex justify="space-between">
        <strong>Status</strong>
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
      </Flex>
      <Divider />
      <Space direction="vertical" size="small" style={{ display: "flex" }}>
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
        {!isLoading && !error && <DriveList stats={stats} />}
      </Space>
    </Card>
  );
}

type DriveListProps = {
  stats: SystemStats;
};

export function DriveList(props: DriveListProps) {
  const formatBytes = (bytes: number, decimals = 2) => {
    if (bytes === 0) return "0 Bytes";

    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
  };

  const formatPercentage = (free: number, total: number) => {
    return Math.round(((total - free) / total) * 100);
  };

  return (
    <Flex gap="large" justify="space-around" wrap>
      {props.stats.drives.map((drive) => {
        return (
          <Tooltip
            title={`Free: ${formatBytes(Number(drive.freeBytes))}`}
            placement={"bottom"}
          >
            <Flex vertical justify="center" gap="small">
              <Badge
                count={driveName(drive.mountPoint)}
                color={"#643f91"}
              >
                <Progress
                  type="dashboard"
                  strokeColor={driveColors}
                  status="normal"
                  percent={formatPercentage(
                    Number(drive.freeBytes),
                    Number(drive.totalBytes)
                  )}
                  percentPosition={{ align: "start", type: "outer" }}
                />
              </Badge>
            </Flex>
          </Tooltip>
        );
      })}
    </Flex>
  );
}

function driveName(mountPoint: string) {
  switch (mountPoint) {
    case "/":
      return "system";
    default:
      return "apps";
  }
}

const driveColors: ProgressProps["strokeColor"] = {
  "0%": "#643f91",
  "75%": "#ffe58f",
  "100%": "#ff4d4f",
};

export function InstalledApplicationsList() {
  const { data, error, isLoading } = useQuery(appsHealthCheck);
  const navigate = useNavigate();

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
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={"No apps installed"}
      >
        <Button type="primary" onClick={() => navigate("/store")}>
          App Store
        </Button>
      </Empty>
    );
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

  var status: "error" | "success" = "success";
  if (props.app.status !== AppStatus.HEALTHY) {
    status = "error";
  }

  return (
    <div style={{ padding: 4, width: 64, textAlign: "center" }}>
      <Badge dot status={status}>
        <Avatar src={app.display?.iconUrl} shape="square" size="large" />
      </Badge>
      <div>{app.display?.name}</div>
    </div>
  );
}
