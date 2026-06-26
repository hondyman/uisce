// @ts-nocheck
import { useState, useEffect } from 'react';
import styles from './StewardGranularityReview.module.css';

const { Text, Paragraph } = Typography;

interface GranularityDefinition {
  node_id: string;
  name: string;
  description: string;
  schema_def: {
    dimension: string;
    interval: string;
    offset_days: number;
    offset_hours: number;
    fiscal_label?: string;
    calendar_type: string;
    week_start_day: string;
    granularity_sql: string;
    owner: string;
    version: string;
    golden_path: boolean;
    review_status: string;
  };
  review_status: string;
  golden_path: boolean;
}

interface StewardGranularityReviewProps {
  granularities?: GranularityDefinition[];
  onAction?: (action: string, granularityId: string, data?: any) => void;
  refreshTrigger?: number;
}

const StewardGranularityReview: React.FC<StewardGranularityReviewProps> = ({
  granularities: propGranularities,
  onAction,
  refreshTrigger
}) => {
  const [granularities, setGranularities] = useState<GranularityDefinition[]>(propGranularities || []);
  const [loading, setLoading] = useState(false);
  const [selectedGranularity, setSelectedGranularity] = useState<GranularityDefinition | null>(null);
  const [detailModalVisible, setDetailModalVisible] = useState(false);

  useEffect(() => {
    if (!propGranularities) {
      fetchGranularities();
    }
  }, [refreshTrigger, propGranularities]);

  useEffect(() => {
    if (propGranularities) {
      setGranularities(propGranularities);
    }
  }, [propGranularities]);

  const fetchGranularities = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/granularities');
      const data = await response.json();
      setGranularities(data.granularities);
    } catch (error) {
      message.error('Failed to fetch granularity definitions');
    } finally {
      setLoading(false);
    }
  };

  const handleAction = async (action: string, granularity: GranularityDefinition) => {
    try {
      let endpoint = '';
      const method = 'POST';
      const body: any = {
        steward_user: 'system', // In a real app, get from auth context
        comment: `Granularity ${action} via steward review`
      };

      switch (action) {
        case 'approve':
          endpoint = `/api/steward/approvals/${granularity.node_id}/approve`;
          body.golden_path = true;
          break;
        case 'reject':
          endpoint = `/api/steward/approvals/${granularity.node_id}/reject`;
          break;
        case 'flag':
          endpoint = `/api/steward/approvals/${granularity.node_id}/flag`;
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

      message.success(`Granularity ${action}d successfully`);
      fetchGranularities();

      if (onAction) {
        onAction(action, granularity.node_id, body);
      }
    } catch (error) {
      message.error(`Failed to ${action} granularity`);
    }
  };

  const showGranularityDetails = (granularity: GranularityDefinition) => {
    setSelectedGranularity(granularity);
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

  const getCalendarTypeColor = (calendarType: string) => {
    switch (calendarType) {
      case 'gregorian':
        return 'blue';
      case 'fiscal':
        return 'green';
      case 'iso_week':
        return 'purple';
      case 'custom':
        return 'orange';
      default:
        return 'default';
    }
  };

  const getActionButtons = (granularity: GranularityDefinition) => (
    <Space size="small">
      <Tooltip title="View Details">
        <Button
          type="text"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => showGranularityDetails(granularity)}
        />
      </Tooltip>
      {granularity.review_status !== 'approved' && (
        <Tooltip title="Approve">
          <Button
            type="primary"
            size="small"
            icon={<CheckCircleOutlined />}
            onClick={() => handleAction('approve', granularity)}
          />
        </Tooltip>
      )}
      {granularity.review_status !== 'rejected' && (
        <Tooltip title="Reject">
          <Button
            danger
            size="small"
            icon={<CloseCircleOutlined />}
            onClick={() => handleAction('reject', granularity)}
          />
        </Tooltip>
      )}
      <Tooltip title="Flag for Review">
        <Button
          size="small"
          icon={<FlagOutlined />}
          onClick={() => handleAction('flag', granularity)}
        />
      </Tooltip>
    </Space>
  );

  return (
    <>
      <Card
        title={`📆 Custom Granularities (${granularities.length})`}
        size="small"
        extra={
          <Button size="small" onClick={fetchGranularities}>
            Refresh
          </Button>
        }
      >
        <List
          loading={loading}
          dataSource={granularities}
    renderItem={(granularity: GranularityDefinition) => (
            <List.Item
              key={granularity.node_id}
              actions={[getActionButtons(granularity)]}
            >
              <List.Item.Meta
                avatar={<ClockCircleOutlined style={{ color: '#52c41a' }} />}
                title={
                  <Space>
                    <span>{granularity.name}</span>
                    <Tag color={getStatusColor(granularity.review_status)}>
                      {granularity.review_status.replace('_', ' ')}
                    </Tag>
                    <Tag color={getCalendarTypeColor(granularity.schema_def.calendar_type)}>
                      {granularity.schema_def.calendar_type}
                    </Tag>
                    {granularity.golden_path && <Tag color="gold">Golden Path</Tag>}
                  </Space>
                }
                description={
                  <div>
                    <Paragraph ellipsis={{ rows: 2, expandable: false }}>
                      {granularity.description}
                    </Paragraph>
                    <Space size="small" wrap>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        Dimension: {granularity.schema_def.dimension}
                      </Text>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        Interval: {granularity.schema_def.interval}
                      </Text>
                      {(granularity.schema_def.offset_days !== 0 || granularity.schema_def.offset_hours !== 0) && (
                        <Text type="secondary" style={{ fontSize: '12px' }}>
                          Offset: {granularity.schema_def.offset_days}d {granularity.schema_def.offset_hours}h
                        </Text>
                      )}
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        Owner: {granularity.schema_def.owner}
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
        title={`📆 Granularity Details: ${selectedGranularity?.name}`}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        width={800}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            Close
          </Button>
        ]}
      >
        {selectedGranularity && (
          <div className={styles.granularityDetails}>
            <Descriptions column={2} size="small">
              <Descriptions.Item label="Node ID">{selectedGranularity.node_id}</Descriptions.Item>
              <Descriptions.Item label="Status">
                <Tag color={getStatusColor(selectedGranularity.review_status)}>
                  {selectedGranularity.review_status.replace('_', ' ')}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Calendar Type">
                <Tag color={getCalendarTypeColor(selectedGranularity.schema_def.calendar_type)}>
                  {selectedGranularity.schema_def.calendar_type}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Week Start">
                {selectedGranularity.schema_def.week_start_day}
              </Descriptions.Item>
              <Descriptions.Item label="Dimension">{selectedGranularity.schema_def.dimension}</Descriptions.Item>
              <Descriptions.Item label="Interval">{selectedGranularity.schema_def.interval}</Descriptions.Item>
              <Descriptions.Item label="Offset Days">{selectedGranularity.schema_def.offset_days}</Descriptions.Item>
              <Descriptions.Item label="Offset Hours">{selectedGranularity.schema_def.offset_hours}</Descriptions.Item>
              <Descriptions.Item label="Owner">{selectedGranularity.schema_def.owner}</Descriptions.Item>
              <Descriptions.Item label="Version">{selectedGranularity.schema_def.version}</Descriptions.Item>
              <Descriptions.Item label="Golden Path">
                {selectedGranularity.golden_path ? 'Yes' : 'No'}
              </Descriptions.Item>
            </Descriptions>

            {selectedGranularity.schema_def.fiscal_label && (
              <div className={styles.fiscalLabel}>
                <h4>Fiscal Label</h4>
                <Text code>{selectedGranularity.schema_def.fiscal_label}</Text>
              </div>
            )}

            <div className={styles.granularitySQL}>
              <h4>Generated SQL</h4>
              <pre className={styles.sqlCode}>
                {selectedGranularity.schema_def.granularity_sql}
              </pre>
            </div>
          </div>
        )}
      </Modal>
    </>
  );
};

export default StewardGranularityReview;
