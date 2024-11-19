import React, { useState } from 'react';
import { useQuery } from '@connectrpc/connect-query';
import type { FormProps } from 'antd';
import {
  DatabaseOutlined,
  FileAddOutlined,
  LoadingOutlined,
  ProfileOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import {
  Divider,
  Row,
  Col,
  Card,
  Form,
  Input,
  Select,
  Upload,
  Spin,
  Alert,
  Flex,
  Button,
} from 'antd';
import { AppStorage } from 'api/platform/server/v1/web_pb';
import { HelpModal } from '../../components/HelpModal';
import { getAppStorage } from 'api/platform/server/v1/web-WebService_connectquery';

const helpInfo = [
  {
    title: 'App volume',
    avatar: <DatabaseOutlined />,
    description:
      "Choose the installed App storage volume you want to upload this file to. For example, you could upload a movie file to your Jellyfin collection by choosing the 'jellyfin-media' option.",
  },
  {
    title: 'File path',
    avatar: <ProfileOutlined />,
    description:
      "Input the path within the selected App's storage to upload the file. For example, if you have a folder called 'movies/family/' in your Jellyfin collection you would type 'movies/family/' here.",
  },
  {
    title: 'Select files',
    avatar: <FileAddOutlined />,
    description:
      'Choose the file(s) to upload. This can be any file you want: videos, music, photos, etc.',
  },
];

export default function UploadPage() {
  return (
    <Flex justify="center">
      <Card bordered={false} style={{ maxWidth: 450, flex: "auto" }}>
        <Row justify={'space-between'}>
          <Col span={7}>
            <strong>Upload Files</strong>
          </Col>
          <HelpModal title="Upload Help" items={helpInfo} />
        </Row>
        <Divider />
        <UploadForm />
      </Card>
    </Flex>
  );
}

type UploadFormFields = {
  volume: string;
  filePath?: string;
  fileName?: string;
  file?: any;
};

function UploadForm() {
  const { data, error, isLoading } = useQuery(getAppStorage);
  const [filePath, setFilePath] = useState('');
  const [volume, setVolume] = useState('');

  var apps: AppStorage[] = [];
  if (data?.apps) {
    apps = data.apps;
  }

  const onFinish: FormProps<UploadFormFields>['onFinish'] = (values) => {
    console.log('Success:', values);
  };

  const onFinishFailed: FormProps<UploadFormFields>['onFinishFailed'] = (
    errorInfo
  ) => {
    console.log('Failed:', errorInfo);
  };

  const normFile = (e: any) => {
    if (Array.isArray(e)) {
      return e;
    }
    return e?.fileList;
  };

  return (
    <>
      {isLoading && <Spin indicator={<LoadingOutlined spin />} size="large" />}
      {error && (
        <Alert
          message="Failed to load app storage"
          description={error.message}
          type="error"
          showIcon
        />
      )}
      {!isLoading && !error && (
        <Form
          name="basic"
          layout="vertical"
          initialValues={{ remember: true }}
          onFinish={onFinish}
          onFinishFailed={onFinishFailed}
          autoComplete="off"
          requiredMark="optional"
        >
          <Form.Item<UploadFormFields>
            label="App volume"
            name="volume"
            rules={[{ required: true, message: 'Please select an app volume' }]}
          >
            <Select onSelect={setVolume}>
              {apps.map((app) => {
                return app.volumes.map((volume) => {
                  return <Select.Option value={volume}>{volume}</Select.Option>;
                });
              })}
            </Select>
          </Form.Item>

          <Form.Item<UploadFormFields> label="File path" name="filePath">
            <Input onChange={(e) => setFilePath(e.target.value)} />
          </Form.Item>

          <Form.Item<UploadFormFields>
            label="Select files"
            name="file"
            valuePropName="fileList"
            getValueFromEvent={normFile}
            rules={[{ required: true, message: 'Please select a file' }]}
          >
            <Upload
              action={`/api/upload`}
              multiple={true}
              method="post"
              listType="picture"
              disabled={volume === ''}
              data={{
                path: filePath,
                volume: volume,
              }}
            >
              <Button
                type="primary"
                icon={<UploadOutlined />}
                disabled={volume === ''}
              >
                Upload
              </Button>
            </Upload>
          </Form.Item>
        </Form>
      )}
    </>
  );
}
