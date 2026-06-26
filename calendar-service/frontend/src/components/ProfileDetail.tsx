import React from 'react';
import { Modal, Descriptions, Tag, Space, Typography, Divider, Empty, Button } from 'antd';
import { CalendarOutlined, ClockCircleOutlined, GlobalOutlined, CopyOutlined } from '@ant-design/icons';
import { message } from 'antd';

const { Text } = Typography;

export interface Profile {
  id: string;
  profile_name: string;
  description?: string;
  calendars: string[];
  conflict_resolution: string;
  timezone: string;
  active: boolean;
  valid_from: string;
  created_at: string;
  updated_at: string;
}

export interface ProfileDetailProps {
  profile: Profile | null;
  visible: boolean;
  onClose: () => void;
}

export const ProfileDetail: React.FC<ProfileDetailProps> = ({
  profile,
  visible,
  onClose,
}) => {
  if (!profile) return null;

  const conflictResolutionColors: Record<string, string> = {
    union: 'green',
    intersection: 'orange',
    priority: 'purple',
  };

  const conflictResolutionDescriptions: Record<string, string> = {
    union: 'Blocked if ANY calendar blocks',
    intersection: 'Blocked only if ALL calendars block',
    priority: 'Highest priority calendar wins',
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    message.success('Copied to clipboard');
  };

  return (
    <Modal
      title={
        <Text strong>Profile Details: {profile.profile_name}</Text>
      }
      open={visible}
      onCancel={onClose}
      footer={null}
      width={900}
      destroyOnClose
    >
      <Descriptions 
        bordered 
        column={{ xxl: 2, xl: 2, lg: 1, md: 1, sm: 1, xs: 1 }}
        size="middle"
      >
        <Descriptions.Item label="Profile ID" span={2}>
          <Space>
            <Text code>{profile.id.substring(0, 8)}...</Text>
            <Button
              type="text"
              size="small"
              icon={<CopyOutlined />}
              onClick={() => copyToClipboard(profile.id)}
              title="Copy full ID"
            />
          </Space>
        </Descriptions.Item>

        <Descriptions.Item label="Profile Name">
          <Text strong>{profile.profile_name}</Text>
        </Descriptions.Item>

        <Descriptions.Item label="Status">
          <Tag color={profile.active ? 'success' : 'default'}>
            {profile.active ? 'Active' : 'Inactive'}
          </Tag>
        </Descriptions.Item>

        <Descriptions.Item label="Description" span={2}>
          {profile.description ? (
            <Text>{profile.description}</Text>
          ) : (
            <Text type="secondary">—</Text>
          )}
        </Descriptions.Item>

        <Descriptions.Item label="Timezone">
          <Space>
            <GlobalOutlined style={{ color: '#1890ff' }} />
            <Tag color="blue">{profile.timezone}</Tag>
          </Space>
        </Descriptions.Item>

        <Descriptions.Item label="Conflict Resolution">
          <Tag color={conflictResolutionColors[profile.conflict_resolution] || 'default'}>
            {profile.conflict_resolution.toUpperCase()}
          </Tag>
        </Descriptions.Item>

        <Descriptions.Item label="Calendars Count" span={2}>
          <Text strong>{profile.calendars.length} calendar{profile.calendars.length !== 1 ? 's' : ''}</Text>
        </Descriptions.Item>

        <Descriptions.Item label="Created" span={2}>
          <Space>
            <ClockCircleOutlined style={{ color: '#1890ff' }} />
            {new Date(profile.created_at).toLocaleString()}
          </Space>
        </Descriptions.Item>

        <Descriptions.Item label="Last Updated" span={2}>
          <Space>
            <ClockCircleOutlined style={{ color: '#1890ff' }} />
            {new Date(profile.updated_at).toLocaleString()}
          </Space>
        </Descriptions.Item>
      </Descriptions>

      <Divider orientation="left">
        <Text strong>Included Calendars</Text>
      </Divider>

      {profile.calendars.length > 0 ? (
        <div style={{ marginBottom: 24 }}>
          {profile.calendars.map((calId, idx) => (
            <Tag
              key={idx}
              icon={<CalendarOutlined />}
              style={{ marginBottom: 8, marginRight: 8, padding: '4px 12px' }}
            >
              {calId}
            </Tag>
          ))}
        </div>
      ) : (
        <Empty description="No calendars" style={{ margin: '16px 0' }} />
      )}

      <Divider orientation="left">
        <Text strong>Conflict Resolution Rules</Text>
      </Divider>

      <Descriptions column={1} size="small">
        <Descriptions.Item label="Current Strategy">
          <Tag color={conflictResolutionColors[profile.conflict_resolution] || 'default'}>
            {profile.conflict_resolution.toUpperCase()}
          </Tag>
        </Descriptions.Item>

        <Descriptions.Item label="Description">
          <Text>{conflictResolutionDescriptions[profile.conflict_resolution]}</Text>
        </Descriptions.Item>

        <Descriptions.Item label="UNION (AND)">
          <Text type="secondary">
            A time slot is blocked if it conflicts with **any** calendar in this profile. Most common for strict compliance and safety-critical operations.
          </Text>
        </Descriptions.Item>

        <Descriptions.Item label="INTERSECTION (OR)">
          <Text type="secondary">
            A time slot is blocked only if it conflicts with **all** calendars. Useful for permissive scheduling where only complete overlap constitutes a block.
          </Text>
        </Descriptions.Item>

        <Descriptions.Item label="PRIORITY">
          <Text type="secondary">
            The calendar with the highest priority determines blocking. Each calendar can have a numeric priority assigned during creation. Useful for hierarchical scheduling decisions.
          </Text>
        </Descriptions.Item>
      </Descriptions>

      <Divider orientation="left">
        <Text strong>About Bitemporal Versioning</Text>
      </Divider>

      <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 16 }}>
        This profile is part of a bitemporal version control system. Each update creates a new version with a new ID, and the previous version's <code>valid_to</code> is set to the update timestamp. All historical versions remain queryable via the <code>/api/v1/profiles/{'{id}'}/versions</code> endpoint.
      </Text>
    </Modal>
  );
};

export default ProfileDetail;
