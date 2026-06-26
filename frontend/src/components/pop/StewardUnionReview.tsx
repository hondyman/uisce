// @ts-nocheck
import React, { useState, useEffect } from 'react';
import styles from './StewardUnionReview.module.css';

const { Text, Paragraph } = Typography;

interface UnionDefinition {
  node_id: string;
  name: string;
  description: string;
  schema_def: {
    source_tables: string[];
    union_type: string;
    union_sql: string;
    table_aliases?: Record<string, string>;
    tags?: string[];
    owner: string;
    version: string;
    golden_path: boolean;
    review_status: string;
  };
  review_status: string;
  golden_path: boolean;
}

interface StewardUnionReviewProps {
  unions?: UnionDefinition[];
  onAction?: (action: string, unionId: string, data?: any) => void;
  refreshTrigger?: number;
}

const StewardUnionReview: React.FC<StewardUnionReviewProps> = ({
  unions: propUnions,
  onAction,
  refreshTrigger
}) => {
  const [unions, setUnions] = useState<UnionDefinition[]>(propUnions || []);
  const [loading, setLoading] = useState(false);
  const [selectedUnion, setSelectedUnion] = useState<UnionDefinition | null>(null);
  const [detailModalVisible, setDetailModalVisible] = useState(false);

  const fetchUnions = React.useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/unions');
      const data = await response.json();
      setUnions(data.unions);
    } catch (error) {
      message.error('Failed to fetch union definitions');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!propUnions) {
      fetchUnions();
    }
  }, [refreshTrigger, propUnions, fetchUnions]);

  useEffect(() => {
    if (propUnions) {
      setUnions(propUnions);
    }
  }, [propUnions]);

  const handleAction = async (action: string, union: UnionDefinition) => {
    try {
      let endpoint = '';
      const method = 'POST';
      const body: any = {
        steward_user: 'system', // In a real app, get from auth context
        comment: `Union ${action} via steward review`
      };

      switch (action) {
        case 'approve':
          endpoint = `/api/steward/approvals/${union.node_id}/approve`;
          body.golden_path = true;
          break;
        case 'reject':
          endpoint = `/api/steward/approvals/${union.node_id}/reject`;
          break;
        case 'flag':
          endpoint = `/api/steward/approvals/${union.node_id}/flag`;
          body.severity = 'medium';
          break;
      }

      const response = await fetch(endpoint, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error('Action failed');
      }

      message.success(`Union ${action}d successfully`);
      fetchUnions();

      if (onAction) {
        onAction(action, union.node_id, body);
      }
    } catch (error) {
      message.error(`Failed to ${action} union`);
    }
  };

  const showUnionDetails = (union: UnionDefinition) => {
    setSelectedUnion(union);
    setDetailModalVisible(true);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'approved':
        return 'green';
      case 'rejected':
        return 'red';
      case 'pending_review':
        return 'orange';
      case 'draft':
        return 'blue';
      default:
        return 'default';
    }
  };

  const getActionButtons = (union: UnionDefinition) => (
    <Space size="small">
      <Tooltip title="View Details">
        <Button
          type="text"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => showUnionDetails(union)}
        />
      </Tooltip>
      {union.review_status !== 'approved' && (
        <Tooltip title="Approve">
          <Button
            type="primary"
            size="small"
            icon={<CheckCircleOutlined />}
            onClick={() => handleAction('approve', union)}
          />
        </Tooltip>
      )}
      {union.review_status !== 'rejected' && (
        <Tooltip title="Reject">
          <Button
            danger
            size="small"
            icon={<CloseCircleOutlined />}
            onClick={() => handleAction('reject', union)}
          />
        </Tooltip>
      )}
      <Tooltip title="Flag for Review">
        <Button
          size="small"
          icon={<FlagOutlined />}
          onClick={() => handleAction('flag', union)}
        />
      </Tooltip>
    </Space>
  );

  return (
    <>
      <Card
        title={`🧩 Dynamic Union Tables (${unions.length})`}
        size="small"
        extra={
          <Button size="small" onClick={fetchUnions}>
            Refresh
          </Button>
        }
      >
        <List
          loading={loading}
          dataSource={unions}
          renderItem={(union: UnionDefinition) => (
            <List.Item
              key={union.node_id}
              actions={[getActionButtons(union)]}
            >
              <List.Item.Meta
                avatar={<DatabaseOutlined style={{ color: '#1890ff' }} />}
                title={
                  <Space>
                    <span>{union.name}</span>
                    <Tag color={getStatusColor(union.review_status)}>
                      {union.review_status.replace('_', ' ')}
                    </Tag>
                    {union.golden_path && <Tag color="gold">Golden Path</Tag>}
                  </Space>
                }
                description={
                  <div>
                    <Paragraph ellipsis={{ rows: 2, expandable: false }}>
                      {union.description}
                    </Paragraph>
                    <Space size="small" wrap>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        Tables: {union.schema_def.source_tables.length}
                      </Text>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        Type: {union.schema_def.union_type}
                      </Text>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        Owner: {union.schema_def.owner}
                      </Text>
                    </Space>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Card>

      <Modal
        title={`🧩 Union Details: ${selectedUnion?.name}`}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        width={800}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            Close
          </Button>
        ]}
      >
        {selectedUnion && (
          <div className={styles.unionDetails}>
            <Descriptions column={2} size="small">
              <Descriptions.Item label="Node ID">{selectedUnion.node_id}</Descriptions.Item>
              <Descriptions.Item label="Status">
                <Tag color={getStatusColor(selectedUnion.review_status)}>
                  {selectedUnion.review_status.replace('_', ' ')}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Owner">{selectedUnion.schema_def.owner}</Descriptions.Item>
              <Descriptions.Item label="Version">{selectedUnion.schema_def.version}</Descriptions.Item>
              <Descriptions.Item label="Union Type">{selectedUnion.schema_def.union_type}</Descriptions.Item>
              <Descriptions.Item label="Golden Path">
                {selectedUnion.golden_path ? 'Yes' : 'No'}
              </Descriptions.Item>
            </Descriptions>

            <div className={styles.sourceTables}>
              <h4>Source Tables ({selectedUnion.schema_def.source_tables.length})</h4>
              <Space wrap>
                {selectedUnion.schema_def.source_tables.map(table => (
                  <Tag key={table} color="blue">{table}</Tag>
                ))}
              </Space>
            </div>

            {selectedUnion.schema_def.table_aliases && Object.keys(selectedUnion.schema_def.table_aliases).length > 0 && (
              <div className={styles.tableAliases}>
                <h4>Table Aliases</h4>
                <Space direction="vertical">
                  {Object.entries(selectedUnion.schema_def.table_aliases).map(([table, alias]) => (
                    <Text key={table} code>{table} → {alias}</Text>
                  ))}
                </Space>
              </div>
            )}

            <div className={styles.unionSQL}>
              <h4>Generated SQL</h4>
              <pre className={styles.sqlCode}>
                {selectedUnion.schema_def.union_sql}
              </pre>
            </div>

            {selectedUnion.schema_def.tags && selectedUnion.schema_def.tags.length > 0 && (
              <div className={styles.tags}>
                <h4>Tags</h4>
                <Space wrap>
                  {selectedUnion.schema_def.tags.map(tag => (
                    <Tag key={tag}>{tag}</Tag>
                  ))}
                </Space>
              </div>
            )}
          </div>
        )}
      </Modal>
    </>
  );
};

export default StewardUnionReview;
