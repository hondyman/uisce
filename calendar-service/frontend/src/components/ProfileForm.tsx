import React, { useState, useEffect } from 'react';
import { useMutation, useQuery, gql } from '@apollo/client';
import {
  Modal, Form, Input, Select, Checkbox, Button, Alert, Tooltip,
  Typography, Space, Tag, Spin, message
} from 'antd';
import { CalendarOutlined, InfoCircleOutlined } from '@ant-design/icons';

const { Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

const CREATE_PROFILE = gql`
  mutation CreateProfile($tenantId: uuid!, $object: schedule_profiles_insert_input!) {
    insert_schedule_profiles_one(object: $object) {
      id
      profile_name
      created_at
    }
  }
`;

const UPDATE_PROFILE = gql`
  mutation UpdateProfile($tenantId: uuid!, $id: uuid!, $object: schedule_profiles_set_input!) {
    update_schedule_profiles_by_pk(
      pk_columns: {id: $id},
      _set: $object
    ) {
      id
      profile_name
      updated_at
    }
  }
`;

const LIST_CALENDARS = gql`
  query ListCalendars($tenantId: uuid!) {
    calendars(
      where: {tenant_id: {_eq: $tenantId}, valid_to: {_is_null: true}},
      order_by: {name: asc}
    ) {
      id
      name
      region
    }
  }
`;

export interface ProfileFormProps {
  tenantId: string;
  visible: boolean;
  onClose: () => void;
  onSuccess: () => void;
  initialValues?: any;
}

export const ProfileForm: React.FC<ProfileFormProps> = ({
  tenantId,
  visible,
  onClose,
  onSuccess,
  initialValues,
}) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const isEditing = !!initialValues;

  const [createProfile] = useMutation(CREATE_PROFILE);
  const [updateProfile] = useMutation(UPDATE_PROFILE);

  const { loading: calendarsLoading, data: calendarsData } = useQuery(LIST_CALENDARS, {
    variables: { tenantId },
    skip: !visible,
    fetchPolicy: 'cache-and-network',
  });

  useEffect(() => {
    if (visible && initialValues) {
      form.setFieldsValue({
        profile_name: initialValues.profile_name,
        description: initialValues.description,
        calendars: initialValues.calendars,
        conflict_resolution: initialValues.conflict_resolution,
        timezone: initialValues.timezone,
        active: initialValues.active,
      });
    } else if (visible) {
      form.resetFields();
      form.setFieldsValue({
        conflict_resolution: 'union',
        timezone: 'UTC',
        active: true,
      });
    }
  }, [visible, initialValues, form]);

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      const object = {
        tenant_id: tenantId,
        profile_name: values.profile_name,
        description: values.description || null,
        calendars: values.calendars,
        conflict_resolution: values.conflict_resolution || 'union',
        timezone: values.timezone || 'UTC',
        active: values.active !== false,
      };

      if (isEditing) {
        await updateProfile({
          variables: {
            tenantId,
            id: initialValues.id,
            object: {
              profile_name: values.profile_name,
              description: values.description || null,
              calendars: values.calendars,
              conflict_resolution: values.conflict_resolution || 'union',
              timezone: values.timezone || 'UTC',
              active: values.active !== false,
            },
          },
        });
      } else {
        await createProfile({ 
          variables: { 
            tenantId,
            object 
          } 
        });
      }

      onSuccess();
    } catch (err: any) {
      console.error('Profile form error:', err);
      message.error(`Failed to ${isEditing ? 'update' : 'create'} profile: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const timezones = [
    'UTC',
    'America/New_York',
    'America/Chicago',
    'America/Denver',
    'America/Los_Angeles',
    'Europe/London',
    'Europe/Paris',
    'Europe/Berlin',
    'Europe/Amsterdam',
    'Asia/Tokyo',
    'Asia/Shanghai',
    'Asia/Singapore',
    'Asia/Hong_Kong',
    'Australia/Sydney',
    'Australia/Melbourne',
  ];

  return (
    <Modal
      title={
        <Text strong>
          {isEditing ? 'Edit Profile (Bitemporal Versioning)' : 'Create New Profile'}
        </Text>
      }
      open={visible}
      onCancel={onClose}
      footer={null}
      width={700}
      destroyOnClose
    >
      {isEditing && (
        <Alert
          message="Note: Updating a profile creates a new version while preserving history"
          description="The old version will have valid_to set to the current timestamp. All versions are queryable via the versions endpoint."
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        disabled={calendarsLoading || loading}
        requiredMark="optional"
      >
        <Form.Item
          name="profile_name"
          label={
            <Space size={4}>
              <span>Profile Name</span>
              <Tooltip title="A unique, descriptive name for this profile (e.g., 'US-Core', 'EU-Finance')">
                <InfoCircleOutlined style={{ color: '#999' }} />
              </Tooltip>
            </Space>
          }
          rules={[
            { required: true, message: 'Profile name is required' },
            { min: 2, message: 'Profile name must be at least 2 characters' },
            { max: 100, message: 'Profile name must be less than 100 characters' },
          ]}
        >
          <Input placeholder="e.g., US-Core, EU-Finance, APAC-Operations" />
        </Form.Item>

        <Form.Item
          name="description"
          label={
            <Space size={4}>
              <span>Description</span>
              <Tooltip title="Optional description of this profile's purpose">
                <InfoCircleOutlined style={{ color: '#999' }} />
              </Tooltip>
            </Space>
          }
        >
          <TextArea 
            rows={2} 
            placeholder="Optional: Describe the purpose, regions, or teams using this profile" 
            maxLength={500}
            showCount
          />
        </Form.Item>

        <Form.Item
          name="calendars"
          label={
            <Space size={4}>
              <span>Calendars</span>
              <Tooltip title="Select one or more calendars to include in this profile">
                <InfoCircleOutlined style={{ color: '#999' }} />
              </Tooltip>
            </Space>
          }
          rules={[
            { 
              validator: (_, value) => {
                if (!value || value.length === 0) {
                  return Promise.reject('At least one calendar is required');
                }
                return Promise.resolve();
              }
            }
          ]}
        >
          <Select
            mode="multiple"
            placeholder="Select calendars"
            loading={calendarsLoading}
            maxTagCount="responsive"
            notFoundContent={calendarsLoading ? <Spin size="small" /> : 'No calendars available'}
          >
            {calendarsData?.calendars?.map((cal: any) => (
              <Option key={cal.id} value={cal.id}>
                <Space>
                  <CalendarOutlined style={{ color: '#1890ff' }} />
                  <strong>{cal.name}</strong>
                  <Tag>{cal.region}</Tag>
                </Space>
              </Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          name="conflict_resolution"
          label={
            <Space size={4}>
              <span>Conflict Resolution</span>
              <Tooltip title="How to handle conflicts when checking availability across multiple calendars">
                <InfoCircleOutlined style={{ color: '#999' }} />
              </Tooltip>
            </Space>
          }
          rules={[{ required: true }]}
        >
          <Select placeholder="Select conflict resolution strategy">
            <Option value="union">
              <div>
                <Tag color="green">UNION</Tag>
                <Text> - Blocked if ANY calendar blocks (most restrictive)</Text>
              </div>
            </Option>
            <Option value="intersection">
              <div>
                <Tag color="orange">INTERSECTION</Tag>
                <Text> - Blocked only if ALL calendars block (most permissive)</Text>
              </div>
            </Option>
            <Option value="priority">
              <div>
                <Tag color="purple">PRIORITY</Tag>
                <Text> - Highest priority calendar wins</Text>
              </div>
            </Option>
          </Select>
        </Form.Item>

        <Form.Item
          name="timezone"
          label={
            <Space size={4}>
              <span>Timezone</span>
              <Tooltip title="The timezone for interpreting time ranges in this profile">
                <InfoCircleOutlined style={{ color: '#999' }} />
              </Tooltip>
            </Space>
          }
          rules={[{ required: true }]}
        >
          <Select 
            showSearch 
            placeholder="Select timezone"
            optionFilterProp="children"
            filterOption={(input, option) =>
              (option?.children as string)?.toLowerCase().includes(input.toLowerCase())
            }
          >
            {timezones.map((tz) => (
              <Option key={tz} value={tz}>
                {tz}
              </Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          name="active"
          label="Active"
          valuePropName="checked"
          initialValue={true}
        >
          <Checkbox>Enable this profile for availability checks</Checkbox>
        </Form.Item>

        <Form.Item>
          <Space>
            <Button 
              htmlType="submit" 
              type="primary" 
              loading={loading}
              size="large"
            >
              {isEditing ? 'Update Profile' : 'Create Profile'}
            </Button>
            <Button onClick={onClose} disabled={loading} size="large">
              Cancel
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ProfileForm;
