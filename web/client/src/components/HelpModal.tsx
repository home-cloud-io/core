import React, { useState, ReactNode } from 'react';

import { List, Col, Modal } from 'antd';
import { QuestionCircleOutlined } from '@ant-design/icons';

type HelpModalItem = {
  title: string;
  avatar: ReactNode;
  description: string;
};

type HelpModalProps = {
  title: string;
  items: HelpModalItem[];
};

export function HelpModal(props: HelpModalProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);

  const showModal = () => {
    setIsModalOpen(true);
  };

  const handleCancel = () => {
    setIsModalOpen(false);
  };

  return (
    <Col span={1}>
      <QuestionCircleOutlined onClick={showModal} />
      <Modal
        title={props.title}
        open={isModalOpen}
        onCancel={handleCancel}
        footer={[]}
      >
        <List
          itemLayout="horizontal"
          dataSource={props.items}
          renderItem={(item, index) => (
            <List.Item>
              <List.Item.Meta
                avatar={item.avatar}
                title={item.title}
                description={item.description}
              />
            </List.Item>
          )}
        />
      </Modal>
    </Col>
  );
}
