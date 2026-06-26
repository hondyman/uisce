// @ts-nocheck
// frontend/src/pages/admin/RelatedObjectsPage.tsx
import { useState, useEffect } from 'react';
import ActionButton from '../../components/ui/ActionButton';
import SVGIcon from '../../components/relationship/SVGIcon';
import { useTenant } from '../../contexts/TenantContext';
import { devError } from '../../utils/devLogger';
import RelatedObjectsPanel from '../../components/catalog/RelatedObjectsPanel';
import { fetchEntitySchema } from '../../api/entitySchema';
import './RelatedObjectsPage.css';

const { Title, Text } = Typography;
const { Option } = Select;

interface Entity {
  id?: string;
  name: string;
  businessName?: string;
  technicalName?: string;
  description?: string;
}

export default function RelatedObjectsPage() {
  const { tenant, datasource } = useTenant();
  const [entities, setEntities] = useState<Entity[]>([]);
  const [selectedEntity, setSelectedEntity] = useState<string>('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadEntities = async () => {
      if (!tenant || !datasource) {
        setLoading(false);
        return;
      }

      try {
        const schema = await fetchEntitySchema(tenant.id, datasource.id || datasource.alpha_tenant_instance_id);
        const entityList = Object.values(schema).map((entity: any) => ({
          id: entity.id, // Capture UUID
          name: entity.businessName || entity.name,
          businessName: entity.businessName,
          technicalName: entity.technicalName,
          description: entity.description,
        }));
        setEntities(entityList);
        if (entityList.length > 0) {
          // Use ID if available, otherwise fallback to name? No, related objects needs ID.
          // entity.id comes from coerceEntitySchema which uses the key (UUID)
          setSelectedEntity(entityList[0].id || entityList[0].name);
        }
      } catch (error) {
        devError('Error loading entities:', error);
        message.error('Failed to load entities');
      } finally {
        setLoading(false);
      }
    };

    loadEntities();
  }, [tenant, datasource]);

  if (loading) {
    return (
      <div className="loadingContainer">
        <Spin size="large" />
      </div>
    );
  }

  if (!tenant || !datasource) {
    return (
      <div className="emptyWrapper">
        <Empty
          description="Please select a tenant and datasource to view related objects"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

  return (
    <div className="pageContainer">
      {/* MIGRATION NOTICE */}
      <Row gutter={16} className="marginBottom24">
        <Col span={24}>
          <Card
            className="migrationCard"
            title={
              <Space>
                <DatabaseOutlined className="dbIcon" />
                <Text strong>Enhanced Integration Available</Text>
              </Space>
            }
          >
            <Space direction="vertical" style={{ width: '100%' }}>
              <Text>
                The Related Objects feature is now integrated directly into the <strong>Entity Manager</strong> (Schema Builder) as a "Relationships" tab.
              </Text>
              <Text type="secondary">
                You can continue using this page, but for the best experience with entity schema management, relationship discovery, and AI suggestions, visit the Entity Manager's Relationships tab.
              </Text>
              <ActionButton
                variant="primary"
                onClick={() => (window.location.href = '/admin/entity-manager')}
              >
                <SVGIcon name="link" className="inline-block mr-2" ariaLabel="go-to-entity-manager" />
                Go to Entity Manager Relationships Tab
              </ActionButton>
            </Space>
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={24}>
          <Card
            title={
              <Space>
                <SVGIcon name="link" className="linkIcon" ariaLabel="related-objects" />
                <span>Related Objects Manager</span>
              </Space>
            }
            extra={
              <Space>
                <RobotOutlined className="robotIcon" />
                <Text type="secondary">AI-Powered Relationship Discovery</Text>
              </Space>
            }
          >
            <Row gutter={16} className="marginBottom24">
              <Col span={24}>
                <Card size="small" className="infoCard">
                  <Space direction="vertical" size="small">
                    <Title level={4} className="titleNoMargin">
                      <DatabaseOutlined className="dbIcon" /> Cross-Entity Relationship Management
                    </Title>
                    <Text>
                      Discover and manage relationships between business objects using AI-powered suggestions
                      and catalog-based semantic mappings. Select an entity below to explore its relationships.
                    </Text>
                  </Space>
                </Card>
              </Col>
            </Row>

            <Row gutter={16} className="marginBottom24">
              <Col span={24}>
                <Space>
                  <Text strong>Select Entity:</Text>
                  <Select
                    value={selectedEntity}
                    onChange={setSelectedEntity}
                    className="selectMinWidth"
                    placeholder="Choose an entity to explore relationships"
                  >
                    {entities.map((entity) => (
                      <Option key={entity.id || entity.name} value={entity.id || entity.name}>
                        {entity.businessName || entity.name}
                        {entity.technicalName && (
                          <Text type="secondary" className="optionTechName">
                            ({entity.technicalName})
                          </Text>
                        )}
                      </Option>
                    ))}
                  </Select>
                </Space>
              </Col>
            </Row>

            {selectedEntity && (
              <Row gutter={16}>
                <Col span={24}>
                  <RelatedObjectsPanel
                    tenantId={tenant.id}
                    datasourceId={datasource.id || datasource.alpha_tenant_instance_id}
                    entity={selectedEntity}
                  />
                </Col>
              </Row>
            )}

            {entities.length === 0 && (
              <Empty
                description="No entities found. Create entities in Entity Manager first."
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}