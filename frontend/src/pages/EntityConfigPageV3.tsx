// @ts-nocheck
import React, { useState, useMemo, useEffect } from 'react';
import {
  Card,
  Button,
  Form,
  Input,
  Select,
  message,
  Modal,
  Col,
  Row,
  Space,
  Tooltip,
  Popconfirm,
  Tag as _Tag,
  Table,
  Tabs as _Tabs,
  Empty,
  Spin,
  Badge,
  Drawer as _Drawer,
  Divider as _Divider,
  List,
  Affix,
  Tree,
  Layout,
  Segmented as _Segmented,
  InputNumber as _InputNumber,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  CopyOutlined as _CopyOutlined,
  SearchOutlined as _SearchOutlined,
  SaveOutlined,
  LinkOutlined as _LinkOutlined,
  LockOutlined as _LockOutlined,
  UnlockOutlined as _UnlockOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  EyeOutlined as _EyeOutlined,
} from '@ant-design/icons';
import { saveEntitySchema, fetchEntitySchema } from '../api/entitySchema';
import type { Entities, Entity as _Entity, Subtype as _Subtype, Field } from '../types/entity-schema';
import { devLog } from '../utils/devLogger';
import { hasTenantScope } from '../utils/tenantScope';
import ProfessionalSearchInput from '../components/common/ProfessionalSearchInput';
import { businessToTechnicalName as _businessToTechnicalName } from '../utils/nameFormatting';
import { useEnhancedSemanticTerms, semanticTermToField, searchSemanticTerms } from '../hooks/useEnhancedSemanticTerms';
import { useTenant } from '../contexts/TenantContext';
import styles from './EntityConfigPageV3.module.css';

const { Sider, Content } = Layout;
const { TextArea: _TextArea } = Input;
const { Option: _Option } = Select;

// Core Business Objects (Workday-style seed data)
// Empty entities object - will be populated entirely from API/database
const INITIAL_ENTITIES: Entities = {};

interface HierarchyNode {
  key: string;
  title: string | React.ReactNode;
  children?: HierarchyNode[];
  data?: { type: 'entity' | 'subtype'; entityKey: string; subtypeKey?: string };
}

interface SelectedNode {
  type: 'entity' | 'subtype';
  entityKey: string;
  subtypeKey?: string;
}

interface EditingField {
  entityKey: string;
  subtypeKey?: string;
  fieldKey?: string;
  level: 'entity' | 'subtype';
}

export default function EntityConfigPageV2() {
  const [entities, setEntities] = useState<Entities>(INITIAL_ENTITIES);
  const [initialEntities, setInitialEntities] = useState<Entities>(INITIAL_ENTITIES);
  const [searchTerm, setSearchTerm] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null);
  const [editingField, setEditingField] = useState<EditingField | null>(null);
  // form instance currently unused in this view; keep it for future work but prefix to silence lint
  const [_form] = Form.useForm();
  const [semanticSearchTerm, setSemanticSearchTerm] = useState('');
  const { tenant, datasource } = useTenant();
  const { semanticTerms, loading: loadingSemanticTerms } = useEnhancedSemanticTerms(datasource?.id);

  // Load saved schema from backend on mount
  useEffect(() => {
    const loadSchema = async () => {
      if (!hasTenantScope()) {
        devLog('[EntityConfigPageV2] No tenant scope, using core BOs');
        return;
      }

      try {
        devLog('[EntityConfigPageV2] Loading schema from backend');
        const savedSchema = await fetchEntitySchema(tenant?.id, datasource?.id || datasource?.alpha_tenant_instance_id);

        if (Object.keys(savedSchema).length > 0) {
          devLog('[EntityConfigPageV2] Schema loaded:', { savedSchema });
          // Use schema directly from API/database (no merging with hardcoded data)
          setInitialEntities(savedSchema);
          setEntities(savedSchema);
        }
      } catch (error) {
        devLog('[EntityConfigPageV2] Error loading schema:', { error });
      }
    };

    loadSchema();
  }, []);

  // Compute changes (delta)
  const computeChanges = useMemo(() => {
    const changed: string[] = [];
    const deleted: string[] = [];

    for (const key of Object.keys(entities)) {
      if (!(key in initialEntities)) {
        changed.push(key);
      } else if (JSON.stringify(entities[key]) !== JSON.stringify(initialEntities[key])) {
        changed.push(key);
      }
    }

    for (const key of Object.keys(initialEntities)) {
      if (!(key in entities)) {
        deleted.push(key);
      }
    }

    return { changed, deleted };
  }, [entities, initialEntities]);

  // Build hierarchy tree for side pane
  const hierarchyTree: HierarchyNode[] = useMemo(() => {
    return Object.entries(entities)
      .filter(([key]) => key.toLowerCase().includes(searchTerm.toLowerCase()))
      .map(([entityKey, entity]) => ({
        key: entityKey,
        title: (
          <Space size="small">
            <Badge color={entity.isCore ? '#1890ff' : '#52c41a'} />
            <span>{entity.businessName || entity.name}</span>
          </Space>
        ),
        data: { type: 'entity' as const, entityKey },
        children: Object.entries(entity.subtypes).map(([subtypeKey, subtype]) => ({
          key: `${entityKey}|${subtypeKey}`,
          title: (
            <Space size="small">
              <Badge color={subtype.isCore ? '#1890ff' : '#52c41a'} />
              <span>{subtype.businessName || subtype.name}</span>
            </Space>
          ),
          data: { type: 'subtype' as const, entityKey, subtypeKey },
        })),
      }));
  }, [entities, searchTerm]);

  const saveAndApply = async () => {
    devLog('[saveAndApply] Saving changes...');
    setIsSaving(true);

    try {
      if (!hasTenantScope()) {
        message.error('Please select a tenant first');
        return;
      }

      const { changed, deleted } = computeChanges;
      const changedEntities = Object.fromEntries(
        changed.map((key) => [key, entities[key]])
      );

      const payload = { changed: changedEntities, deleted };
      await saveEntitySchema(payload, tenant?.id, datasource?.id);

      setInitialEntities(entities);
      message.success(`✅ Saved! ${changed.length} changed, ${deleted.length} deleted`);
    } catch (error) {
      devLog('[saveAndApply] Error:', { error });
      message.error('Failed to save schema');
    } finally {
      setIsSaving(false);
    }
  };

  // Get selected entity/subtype
  const getSelectedContent = () => {
    if (!selectedNode) return null;

    const entity = entities[selectedNode.entityKey];
    if (!entity) return null;

    if (selectedNode.type === 'entity') {
      return { entity, subtype: null, entityKey: selectedNode.entityKey };
    } else {
      const subtype = entity.subtypes[selectedNode.subtypeKey!];
      return { entity, subtype, entityKey: selectedNode.entityKey, subtypeKey: selectedNode.subtypeKey };
    }
  };

  const content = getSelectedContent();
  const currentFields = content?.subtype?.subtype_fields || content?.entity?.entity_fields || [];

  // Separate inherited and assigned fields
  const inheritedFields = currentFields.filter((f) => f.isCore);
  const assignedFields = currentFields.filter((f) => !f.isCore).sort((a, b) => (a.sequence || 0) - (b.sequence || 0));

  // Filtered semantic terms
  const filteredSemanticTerms = useMemo(
    () => searchSemanticTerms(semanticTerms, semanticSearchTerm),
    [semanticTerms, semanticSearchTerm]
  );

  // Handle add field
  const handleAddField = (semanticTerm: any) => {
    if (!content) return;

    const newField = semanticTermToField(semanticTerm, (assignedFields.length || 0) + inheritedFields.length) as Field;

    const updatedEntity = { ...content.entity };

    if (selectedNode!.type === 'entity') {
      updatedEntity.entity_fields = [...updatedEntity.entity_fields, newField];
      updatedEntity.customFields = [...(updatedEntity.customFields || []), newField];
    } else {
      const updatedSubtypes = { ...updatedEntity.subtypes };
      updatedSubtypes[selectedNode!.subtypeKey!] = {
        ...updatedSubtypes[selectedNode!.subtypeKey!],
        subtype_fields: [...updatedSubtypes[selectedNode!.subtypeKey!].subtype_fields, newField],
      };
      updatedEntity.subtypes = updatedSubtypes;
    }

    setEntities({ ...entities, [selectedNode!.entityKey]: updatedEntity });
    message.success(`Field "${newField.businessName}" added`);
    setSemanticSearchTerm('');
  };

  // Handle delete field
  const handleDeleteField = (fieldKey: string) => {
    if (!content) return;

    const updatedEntity = { ...content.entity };

    if (selectedNode!.type === 'entity') {
      updatedEntity.entity_fields = updatedEntity.entity_fields.filter((f) => f.key !== fieldKey);
      updatedEntity.customFields = updatedEntity.customFields?.filter((f) => f.key !== fieldKey);
    } else {
      const updatedSubtypes = { ...updatedEntity.subtypes };
      updatedSubtypes[selectedNode!.subtypeKey!] = {
        ...updatedSubtypes[selectedNode!.subtypeKey!],
        subtype_fields: updatedSubtypes[selectedNode!.subtypeKey!].subtype_fields.filter((f) => f.key !== fieldKey),
      };
      updatedEntity.subtypes = updatedSubtypes;
    }

    setEntities({ ...entities, [selectedNode!.entityKey]: updatedEntity });
    message.success('Field deleted');
  };

  // Handle reorder field
  const handleReorderField = (fieldKey: string, direction: 'up' | 'down') => {
    if (!content) return;

    const fields = [...assignedFields];
    const idx = fields.findIndex((f) => f.key === fieldKey);

    if (idx === -1) return;
    if ((direction === 'up' && idx === 0) || (direction === 'down' && idx === fields.length - 1)) return;

    const newIdx = direction === 'up' ? idx - 1 : idx + 1;
    [fields[idx], fields[newIdx]] = [fields[newIdx], fields[idx]];

    // Update sequence numbers
    fields.forEach((f, i) => {
      f.sequence = i;
    });

    const updatedEntity = { ...content.entity };

    if (selectedNode!.type === 'entity') {
      updatedEntity.entity_fields = [
        ...inheritedFields.map((f, i) => ({ ...f, sequence: i })),
        ...fields,
      ];
      updatedEntity.customFields = fields;
    } else {
      const allFields = [
        ...inheritedFields.map((f, i) => ({ ...f, sequence: i })),
        ...fields,
      ];
      const updatedSubtypes = { ...updatedEntity.subtypes };
      updatedSubtypes[selectedNode!.subtypeKey!] = {
        ...updatedSubtypes[selectedNode!.subtypeKey!],
        subtype_fields: allFields,
      };
      updatedEntity.subtypes = updatedSubtypes;
    }

    setEntities({ ...entities, [selectedNode!.entityKey]: updatedEntity });
  };

  return (
    <div className={styles.container}>
      {/* HEADER */}
      <Row gutter={16} style={{ marginBottom: '24px' }}>
        <Col span={24}>
          <Card
            title={
              <Space>
                <Badge status="processing" text="Definitions" />
                <span>Entity Schema Builder (Semantic-Driven)</span>
              </Space>
            }
            extra={
              <Button
                type="primary"
                icon={<SaveOutlined />}
                onClick={saveAndApply}
                loading={isSaving}
                disabled={computeChanges.changed.length === 0 && computeChanges.deleted.length === 0}
              >
                SAVE & APPLY ({computeChanges.changed.length + computeChanges.deleted.length})
              </Button>
            }
          >
            <ProfessionalSearchInput
              value={searchTerm}
              onChange={setSearchTerm}
              placeholder="Search entities and subtypes..."
              onClear={() => setSearchTerm('')}
            />
          </Card>
        </Col>
      </Row>

      {/* MAIN LAYOUT: SIDE PANE + CONTENT */}
      <Layout className={styles.layoutContainer}>
        {/* LEFT SIDE PANE: HIERARCHY */}
        <Sider width={300} className={styles.layoutSider}>
          <div className={styles.sidePane}>
            <h3>📋 Hierarchy</h3>
            {hierarchyTree.length === 0 ? (
              <Empty description="No entities" />
            ) : (
              <Tree
                treeData={hierarchyTree}
                onSelect={(selectedKeys) => {
                  if (selectedKeys.length === 0) {
                    setSelectedNode(null);
                    return;
                  }

                  const key = selectedKeys[0] as string;
                  if (key.includes('|')) {
                    const [entityKey, subtypeKey] = key.split('|');
                    setSelectedNode({ type: 'subtype', entityKey, subtypeKey });
                  } else {
                    setSelectedNode({ type: 'entity', entityKey: key });
                  }
                }}
              />
            )}
          </div>
        </Sider>

        {/* RIGHT CONTENT PANE */}
        <Content className={styles.layoutContent}>
          {!selectedNode ? (
            <Empty description="Select an entity or subtype to view fields" />
          ) : (
            <Space direction="vertical" style={{ width: '100%' }}>
              {/* HEADER */}
              <Card size="small">
                <Row gutter={16}>
                  <Col span={24}>
                    <h2 className={styles.contentHeader}>
                      {selectedNode.type === 'entity'
                        ? `${content?.entity?.businessName || content?.entity?.name}`
                        : `${content?.entity?.businessName || content?.entity?.name} → ${content?.subtype?.businessName || content?.subtype?.name}`}
                    </h2>
                  </Col>
                </Row>
              </Card>

              {/* INHERITED FIELDS */}
              {inheritedFields.length > 0 && (
                <Card size="small" title={`🔒 Inherited Fields (${inheritedFields.length})`}>
                  <Table
                    columns={[
                      { dataIndex: 'businessName', title: 'Business Name', key: 'businessName', width: '30%' },
                      { dataIndex: 'technicalName', title: 'Technical Name', key: 'technicalName', width: '30%', render: (text) => <code>{text}</code> },
                      { dataIndex: 'type', title: 'Type', key: 'type', width: '20%' },
                      { dataIndex: 'semanticTermName', title: 'Semantic Term', key: 'semanticTermName', width: '20%' },
                    ]}
                    dataSource={inheritedFields}
                    pagination={false}
                    size="small"
                    rowKey="key"
                  />
                </Card>
              )}

              {/* ASSIGNED FIELDS */}
              <Card
                size="small"
                title={`✏️ Assigned Fields (${assignedFields.length})`}
                extra={
                  <Button
                    type="primary"
                    size="small"
                    icon={<PlusOutlined />}
                    onClick={() => setEditingField({ entityKey: selectedNode.entityKey, subtypeKey: selectedNode.subtypeKey, level: selectedNode.type })}
                  >
                    Add Field
                  </Button>
                }
              >
                {assignedFields.length === 0 ? (
                  <Empty description="No assigned fields. Click 'Add Field' to create one." />
                ) : (
                  <Table
                    columns={[
                      { dataIndex: 'businessName', title: 'Business Name', key: 'businessName', width: '20%' },
                      { dataIndex: 'technicalName', title: 'Technical Name', key: 'technicalName', width: '20%', render: (text) => <code>{text}</code> },
                      { dataIndex: 'type', title: 'Type', key: 'type', width: '15%' },
                      { dataIndex: 'semanticTermName', title: 'Semantic Term', key: 'semanticTermName', width: '25%' },
                      {
                        title: 'Actions',
                        key: 'actions',
                        width: '20%',
                        render: (_, record: Field) => (
                          <Space>
                            <Tooltip title="Move up">
                              <Button
                                size="small"
                                type="text"
                                icon={<ArrowUpOutlined />}
                                onClick={() => handleReorderField(record.key, 'up')}
                                disabled={assignedFields[0]?.key === record.key}
                              />
                            </Tooltip>
                            <Tooltip title="Move down">
                              <Button
                                size="small"
                                type="text"
                                icon={<ArrowDownOutlined />}
                                onClick={() => handleReorderField(record.key, 'down')}
                                disabled={assignedFields[assignedFields.length - 1]?.key === record.key}
                              />
                            </Tooltip>
                            <Tooltip title="Edit">
                              <Button size="small" type="text" icon={<EditOutlined />} disabled />
                            </Tooltip>
                            <Popconfirm title="Delete field?" onConfirm={() => handleDeleteField(record.key)}>
                              <Tooltip title="Delete">
                                <DeleteOutlined style={{ color: '#ff4d4f', cursor: 'pointer' }} />
                              </Tooltip>
                            </Popconfirm>
                          </Space>
                        ),
                      },
                    ]}
                    dataSource={assignedFields}
                    pagination={false}
                    size="small"
                    rowKey="key"
                  />
                )}
              </Card>
            </Space>
          )}
        </Content>
      </Layout>

      {/* MODAL: Add Field (Semantic Term Selection) */}
      <Modal
        title="Add Field - Select Semantic Term"
        open={editingField !== null}
        onCancel={() => setEditingField(null)}
        width={800}
        footer={null}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Input.Search
            placeholder="Search semantic terms..."
            value={semanticSearchTerm}
            onChange={(e) => setSemanticSearchTerm(e.target.value)}
            loading={loadingSemanticTerms}
          />

          {loadingSemanticTerms ? (
            <Spin />
          ) : filteredSemanticTerms.length === 0 ? (
            <Empty description="No semantic terms found" />
          ) : (
            <List
              dataSource={filteredSemanticTerms}
              renderItem={(term) => (
                <List.Item
                  extra={
                    <Button
                      type="primary"
                      onClick={() => {
                        handleAddField(term);
                        setEditingField(null);
                      }}
                    >
                      Add
                    </Button>
                  }
                >
                  <List.Item.Meta
                    title={term.businessName}
                    description={
                      <Space direction="vertical" size={0}>
                        <span>
                          <strong>Technical:</strong> <code>{term.technicalName}</code>
                        </span>
                        <span>
                          <strong>Type:</strong> {term.dataType}
                        </span>
                        {term.description && <span>{term.description}</span>}
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          )}
        </Space>
      </Modal>

      {/* SAVE BUTTON AFFIX */}
      {(computeChanges.changed.length > 0 || computeChanges.deleted.length > 0) && (
        <Affix style={{ bottom: 24, right: 24 }}>
          <Button
            type="primary"
            size="large"
            icon={<SaveOutlined />}
            loading={isSaving}
            onClick={saveAndApply}
            style={{ fontSize: '16px', padding: '12px 24px' }}
          >
            SAVE & APPLY
          </Button>
        </Affix>
      )}
    </div>
  );
}
