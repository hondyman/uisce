import React, { useState, useEffect } from 'react';
import { useQuery, useMutation, gql } from '@apollo/client';
import {
  Table, Button, Space, Tag, Typography, Card, Modal,
  message, Popconfirm, Empty, Spin, Pagination
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, ReloadOutlined } from '@ant-design/icons';
import { ProfileForm } from './ProfileForm';
import { ProfileDetail } from './ProfileDetail';

const { Text } = Typography;

const LIST_PROFILES = gql`
  query ListProfiles($tenantId: uuid!, $limit: Int!, $offset: Int!) {
    schedule_profiles(
      where: {
        tenant_id: {_eq: $tenantId},
        valid_to: {_is_null: true},
        active: {_eq: true}
      },
      limit: $limit,
      offset: $offset,
      order_by: {created_at: desc}
    ) {
      id
      profile_name
      description
      calendars
      conflict_resolution
      timezone
      active
      valid_from
      created_at
      updated_at
    }
  }
`;

const DELETE_PROFILE = gql`
  mutation DeleteProfile($id: uuid!) {
    update_schedule_profiles_by_pk(
      pk_columns: {id: $id},
      _set: {valid_to: "now()", active: false}
    ) {
      id
    }
  }
`;

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

export interface ProfileListProps {
  tenantId: string;
}

export const ProfileList: React.FC<ProfileListProps> = ({ tenantId }) => {
  const [formVisible, setFormVisible] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedProfile, setSelectedProfile] = useState<Profile | null>(null);
  const [editingProfile, setEditingProfile] = useState<Profile | null>(null);
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10 });

  const { loading, error, data, refetch } = useQuery(LIST_PROFILES, {
    variables: { 
      tenantId, 
      limit: pagination.pageSize, 
      offset: (pagination.current - 1) * pagination.pageSize 
    },
    fetchPolicy: 'cache-and-network',
  });

  const [deleteProfile] = useMutation(DELETE_PROFILE, {
    onCompleted: () => {
      message.success('Profile deleted successfully');
      refetch();
    },
    onError: (err) => {
      message.error(`Failed to delete: ${err.message}`);
    },
  });

  const handleDelete = (id: string) => {
    deleteProfile({ variables: { id } });
  };

  const handleEdit = (profile: Profile) => {
    setEditingProfile(profile);
    setFormVisible(true);
  };

  const handleView = (profile: Profile) => {
    setSelectedProfile(profile);
    setDetailVisible(true);
  };

  const handlePaginationChange = (page: number, pageSize: number) => {
    setPagination({ current: page, pageSize });
  };

  const columns = [
    {
      title: 'Profile Name',
      dataIndex: 'profile_name',
      key: 'profile_name',
      width: 180,
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 200,
    },
    {
      title: 'Timezone',
      dataIndex: 'timezone',
      key: 'timezone',
      width: 120,
      render: (tz: string) => <Tag color="blue">{tz}</Tag>,
    },
    {
      title: 'Conflict Resolution',
      dataIndex: 'conflict_resolution',
      key: 'conflict_resolution',
      width: 140,
      render: (cr: string) => {
        const colors: Record<string, string> = {
          union: 'green',
          intersection: 'orange',
          priority: 'purple',
        };
        return <Tag color={colors[cr] || 'default'}>{cr.toUpperCase()}</Tag>;
      },
    },
    {
      title: 'Calendars',
      dataIndex: 'calendars',
      key: 'calendars',
      width: 100,
      render: (cals: string[]) => (
        <Text type="secondary">{cals.length} calendar{cals.length !== 1 ? 's' : ''}</Text>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'active',
      key: 'active',
      width: 80,
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'red'}>
          {active ? 'Active' : 'Inactive'}
        </Tag>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 120,
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      fixed: 'right' as const,
      render: (_: any, record: Profile) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleView(record)}
            title="View details"
          />
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
            title="Edit profile"
          />
          <Popconfirm
            title="Delete Profile"
            description="Are you sure you want to delete this profile? This action cannot be undone."
            onConfirm={() => handleDelete(record.id)}
            okText="Yes"
            cancelText="No"
            okButtonProps={{ danger: true }}
          >
            <Button type="text" size="small" danger icon={<DeleteOutlined />} title="Delete profile" />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  if (error) {
    return (
      <Card>
        <Empty 
          description="Error loading profiles" 
          style={{ marginTop: 48, marginBottom: 48 }}
        />
        <Button onClick={() => refetch()}>Retry</Button>
      </Card>
    );
  }

  const profiles = data?.schedule_profiles || [];

  return (
    <Card
      title={<Text strong>Schedule Profiles</Text>}
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => refetch()}
            loading={loading}
          />
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingProfile(null);
              setFormVisible(true);
            }}
          >
            New Profile
          </Button>
        </Space>
      }
    >
      <Spin spinning={loading && profiles.length === 0}>
        <Table
          dataSource={profiles}
          columns={columns}
          rowKey="id"
          pagination={false}
          scroll={{ x: 1200 }}
        />
        {profiles.length > 0 && (
          <Pagination
            current={pagination.current}
            pageSize={pagination.pageSize}
            total={profiles.length * 2} // Estimate for UI
            onChange={handlePaginationChange}
            showSizeChanger
            pageSizeOptions={['10', '25', '50']}
            style={{ marginTop: 16, textAlign: 'right' }}
          />
        )}
        {!loading && profiles.length === 0 && (
          <Empty
            description="No profiles yet"
            style={{ marginTop: 48, marginBottom: 48 }}
            extra={
              <Button
                type="primary"
                onClick={() => {
                  setEditingProfile(null);
                  setFormVisible(true);
                }}
              >
                Create First Profile
              </Button>
            }
          />
        )}
      </Spin>

      <ProfileForm
        tenantId={tenantId}
        visible={formVisible}
        onClose={() => {
          setFormVisible(false);
          setEditingProfile(null);
        }}
        onSuccess={() => {
          setFormVisible(false);
          setEditingProfile(null);
          refetch();
          message.success(editingProfile ? 'Profile updated successfully' : 'Profile created successfully');
        }}
        initialValues={editingProfile || undefined}
      />

      <ProfileDetail
        profile={selectedProfile}
        visible={detailVisible}
        onClose={() => {
          setDetailVisible(false);
          setSelectedProfile(null);
        }}
      />
    </Card>
  );
};

export default ProfileList;
