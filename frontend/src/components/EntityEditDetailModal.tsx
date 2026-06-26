// @ts-nocheck
import React, { useState, useEffect } from 'react';
import {
  Modal,
  Tree,
  Layout,
  Button,
  Table,
  Space,
  Tag,
  message,
  Popconfirm,
  Tooltip,
  Input,
  Tabs,
  Spin,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  SaveOutlined,
  LockOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import type { Entities, Entity, Field } from '../types/entity-schema';
import { useEnhancedSemanticTerms, semanticTermToField, searchSemanticTerms } from '../hooks/useEnhancedSemanticTerms';
import type { EnhancedSemanticTerm } from '../hooks/useEnhancedSemanticTerms';
import {
  generateCoreModel,
  generateCoreView,
  generateCustomModel,
  generateCustomView,
} from '../api/models';
import styles from './EntityEditDetailModal.module.css';

const { Sider, Content } = Layout;
const { TabPane } = Tabs;

interface EntityEditDetailModalProps {
  visible: boolean;
  entityKey: string;
  entities: Entities;
  onClose: () => void;
  onSave: (entityKey: string, updatedEntity: Entity) => void;
  datasourceId?: string;
}

interface SelectedNode {
  type: 'entity' | 'subtype';
  entityKey: string;
  subtypeKey?: string;
}

interface HierarchyNode {
  title: string | React.ReactNode;
  key: string;
  children?: HierarchyNode[];
}

export default function EntityEditDetailModal({
  visible,
  entityKey,
  entities,
  onClose,
  onSave,
  datasourceId,
}: EntityEditDetailModalProps) {
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null);
  const [editingEntity, setEditingEntity] = useState<Entity | null>(null);
  const [showSemanticModal, setShowSemanticModal] = useState(false);
  const [semanticSearchTerm, setSemanticSearchTerm] = useState('');
  const [selectedFieldTarget, setSelectedFieldTarget] = useState<{
    subtypeKey?: string;
  } | null>(null);
  const [coreModel, setCoreModel] = useState<any>(null);
  const [coreView, setCoreView] = useState<any>(null);
  const [customModel, setCustomModel] = useState<any>(null);
  const [customView, setCustomView] = useState<any>(null);
  const [loadingModel, setLoadingModel] = useState(false);
  const [loadingView, setLoadingView] = useState(false);
  const [loadingCustomModel, setLoadingCustomModel] = useState(false);
  const [loadingCustomView, setLoadingCustomView] = useState(false);

  const { semanticTerms, loading: _termsLoading } = useEnhancedSemanticTerms(datasourceId);

  // silence eslint for imports that may be used in templates or future extensions
  void _termsLoading;

  // Initialize editing entity when modal opens
  useEffect(() => {
    if (visible && entityKey && entities[entityKey]) {
      setEditingEntity(JSON.parse(JSON.stringify(entities[entityKey])));
      // Auto-select the entity
      setSelectedNode({ type: 'entity', entityKey });
      setCoreModel(null);
      setCoreView(null);
    }
  }, [visible, entityKey, entities]);

  if (!visible || !editingEntity) {
    return null;
  }

  const entity = editingEntity;

  // Build hierarchy tree
  const hierarchyTree: HierarchyNode[] = [
    {
      title: (
        <span>
          <Tag color="blue">core</Tag>
          <strong>{entity.businessName || entity.name}</strong>
        </span>
      ),
      key: `entity-${entityKey}`,
      children: entity.subtypes
        ? Object.entries(entity.subtypes).map(([subtypeKey, subtype]) => ({
            title: (
              <span>
                <Tag color={subtype.isCore ? 'blue' : 'green'}>
                  {subtype.isCore ? 'core' : 'custom'}
                </Tag>
                {subtype.businessName || subtype.name}
              </span>
            ),
            key: `subtype-${entityKey}-${subtypeKey}`,
          }))
        : [],
    },
  ];

  // Get selected entity/subtype for field display
  const getSelectedFields = () => {
    if (!selectedNode) return { inherited: [], assigned: [] };

    if (selectedNode.type === 'entity') {
      return {
        inherited: [],
        assigned: entity.entity_fields || [],
      };
    }

    if (selectedNode.type === 'subtype' && selectedNode.subtypeKey) {
      const subtype = entity.subtypes?.[selectedNode.subtypeKey];
      if (!subtype) return { inherited: [], assigned: [] };

      return {
        inherited: entity.entity_fields || [],
        assigned: subtype.subtype_fields || [],
      };
    }

    return { inherited: [], assigned: [] };
  };

  const { inherited, assigned } = getSelectedFields();

  const handleTreeSelect: TreeProps['onSelect'] = (selectedKeys) => {
    if (selectedKeys.length === 0) return;

    const key = selectedKeys[0] as string;
    if (key.startsWith('entity-')) {
      setSelectedNode({ type: 'entity', entityKey });
    } else if (key.startsWith('subtype-')) {
      const parts = key.replace('subtype-', '').split('-');
      const subtypeKey = parts[parts.length - 1];
      setSelectedNode({ type: 'subtype', entityKey, subtypeKey });
    }
  };

  const handleAddField = (subtypeKey?: string) => {
    setSelectedFieldTarget({ subtypeKey });
    setShowSemanticModal(true);
  };

  const handleSelectSemanticTerm = (termId: string) => {
    const term = semanticTerms.find((t: EnhancedSemanticTerm) => t.id === termId);
    if (!term) return;

    const newField = semanticTermToField(term, 0);
    const updated = JSON.parse(JSON.stringify(editingEntity));

    if (!selectedFieldTarget?.subtypeKey) {
      // Adding to entity fields
      if (!updated.entity_fields) updated.entity_fields = [];
      updated.entity_fields.push(newField);
    } else {
      // Adding to subtype fields
      const subtypeKey = selectedFieldTarget.subtypeKey;
      if (!updated.subtypes[subtypeKey].subtype_fields) {
        updated.subtypes[subtypeKey].subtype_fields = [];
      }
      updated.subtypes[subtypeKey].subtype_fields.push(newField);
    }

    setEditingEntity(updated);
    setShowSemanticModal(false);
    message.success('Field added');
  };

  const handleDeleteField = (fieldKey: string, target?: string) => {
    const updated = JSON.parse(JSON.stringify(editingEntity));

    if (!target) {
      // Delete from entity fields
      updated.entity_fields = (updated.entity_fields || []).filter(
        (f: Field) => f.key !== fieldKey
      );
    } else {
      // Delete from subtype fields
      const subtype = updated.subtypes?.[target];
      if (subtype) {
        subtype.subtype_fields = (subtype.subtype_fields || []).filter(
          (f: Field) => f.key !== fieldKey
        );
      }
    }

    setEditingEntity(updated);
    message.success('Field deleted');
  };

  const handleMoveField = (index: number, direction: 'up' | 'down', target?: string) => {
    const updated = JSON.parse(JSON.stringify(editingEntity));
    const newIndex = direction === 'up' ? index - 1 : index + 1;

    if (!target) {
      // Move in entity fields
      const fields = updated.entity_fields || [];
      [fields[index], fields[newIndex]] = [fields[newIndex], fields[index]];
    } else {
      // Move in subtype fields
      const subtype = updated.subtypes?.[target];
      if (subtype) {
        const fields = subtype.subtype_fields || [];
        [fields[index], fields[newIndex]] = [fields[newIndex], fields[index]];
      }
    }

    setEditingEntity(updated);
  };

  const handleSave = () => {
    onSave(entityKey, editingEntity);
    onClose();
  };

  const handleGenerateCoreModel = async () => {
    if (!datasourceId) return;
    setLoadingModel(true);
    try {
      const result = await generateCoreModel(datasourceId, entityKey, entity.name);
      if (result.success) {
        setCoreModel(result.model);
        message.success('Core model generated successfully');
      } else {
        message.error('Failed to generate core model');
      }
    } catch (error) {
      message.error('An error occurred while generating the core model');
    } finally {
      setLoadingModel(false);
    }
  };

  const handleGenerateCoreView = async () => {
    if (!datasourceId || !coreModel) return;
    setLoadingView(true);
    try {
      const result = await generateCoreView(datasourceId, entityKey, coreModel.name);
      if (result.success) {
        setCoreView(result.view);
        message.success('Core view generated successfully');
      } else {
        message.error('Failed to generate core view');
      }
    } catch (error) {
      message.error('An error occurred while generating the core view');
    } finally {
      setLoadingView(false);
    }
  };

  const handleGenerateCustomModel = async () => {
    if (!datasourceId || !coreModel) return;
    setLoadingCustomModel(true);
    try {
      const result = await generateCustomModel(
        datasourceId,
        entityKey,
        entity.name,
        coreModel.name
      );
      if (result.success) {
        setCustomModel(result.model);
        message.success('Custom model generated successfully');
      } else {
        message.error('Failed to generate custom model');
      }
    } catch (error) {
      message.error('An error occurred while generating the custom model');
    } finally {
      setLoadingCustomModel(false);
    }
  };

  const handleGenerateCustomView = async () => {
    if (!datasourceId || !customModel) return;
    setLoadingCustomView(true);
    try {
      const result = await generateCustomView(
        datasourceId,
        entityKey,
        customModel.name
      );
      if (result.success) {
        setCustomView(result.view);
        message.success('Custom view generated successfully');
      } else {
        message.error('Failed to generate custom view');
      }
    } catch (error) {
      message.error('An error occurred while generating the custom view');
    } finally {
      setLoadingCustomView(false);
    }
  };

  // Column definitions for field tables
  const fieldColumns = [
    {
      title: 'Business Name',
      dataIndex: 'businessName',
      key: 'businessName',
      width: 150,
    },
    {
      title: 'Technical Name',
      dataIndex: 'technicalName',
      key: 'technicalName',
      width: 150,
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 100,
    },
    {
      title: 'Semantic Term',
      dataIndex: 'semanticTermName',
      key: 'semanticTermName',
      width: 150,
    },
  ];

  const assignedFieldColumns = [
    ...fieldColumns,
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_: any, record: Field, index: number) => (
        <Space size="small">
          <Tooltip title="Move up">
            <Button
              type="text"
              size="small"
              icon={<ArrowUpOutlined />}
              disabled={index === 0}
              onClick={() =>
                handleMoveField(
                  index,
                  'up',
                  selectedNode?.type === 'subtype' ? selectedNode.subtypeKey : undefined
                )
              }
            />
          </Tooltip>
          <Tooltip title="Move down">
            <Button
              type="text"
              size="small"
              icon={<ArrowDownOutlined />}
              disabled={index === assigned.length - 1}
              onClick={() =>
                handleMoveField(
                  index,
                  'down',
                  selectedNode?.type === 'subtype' ? selectedNode.subtypeKey : undefined
                )
              }
            />
          </Tooltip>
          <Popconfirm
            title="Delete field?"
            onConfirm={() =>
              handleDeleteField(
                record.key,
                selectedNode?.type === 'subtype' ? selectedNode.subtypeKey : undefined
              )
            }
          >
            <Button type="text" danger size="small" icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Modal
        title={`Edit: ${entity.businessName || entity.name}`}
        visible={visible}
        onCancel={onClose}
        width={1200}
        footer={[
          <Button key="cancel" onClick={onClose}>
            Cancel
          </Button>,
          <Button key="save" type="primary" icon={<SaveOutlined />} onClick={handleSave}>
            Save Changes
          </Button>,
        ]}
      >
        <Layout style={{ height: '600px', border: '1px solid #f0f0f0' }}>
          <Sider width={300} style={{ background: '#fafafa' }}>
            <div className={styles.sidePane}>
              <h4>Entity Hierarchy</h4>
              <Tree
                treeData={hierarchyTree}
                onSelect={handleTreeSelect}
                defaultSelectedKeys={[`entity-${entityKey}`]}
              />
            </div>
          </Sider>
          <Content style={{ padding: '16px', overflow: 'auto' }}>
            <Tabs defaultActiveKey="1">
              <TabPane tab="Fields" key="1">
                {selectedNode ? (
                  <>
                    <h3>
                      {selectedNode.type === 'entity'
                        ? `${entity.businessName || entity.name} Fields`
                        : `${entity.subtypes?.[selectedNode.subtypeKey!]?.businessName || entity.subtypes?.[selectedNode.subtypeKey!]?.name} Fields`}
                    </h3>

                    {inherited.length > 0 && (
                      <div className={styles.inheritedSection}>
                        <h4>
                          <LockOutlined /> Inherited Fields ({inherited.length})
                        </h4>
                        <Table
                          columns={fieldColumns}
                          dataSource={inherited.map((f) => ({ ...f, key: f.key }))}
                          pagination={false}
                          size="small"
                          rowClassName={() => 'inherited-row'}
                        />
                      </div>
                    )}

                    <h4>
                      Assigned Fields ({assigned.length}){' '}
                      <Button
                        type="primary"
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={() => handleAddField(selectedNode.subtypeKey)}
                      >
                        Add
                      </Button>
                    </h4>
                    <Table
                      columns={assignedFieldColumns}
                      dataSource={assigned.map((f) => ({ ...f, key: f.key }))}
                      pagination={false}
                      size="small"
                      rowClassName={() => 'assigned-row'}
                    />
                  </>
                ) : (
                  <div className={styles.emptyState}>
                    Select an entity or subtype from the tree to view fields
                  </div>
                )}
              </TabPane>
              <TabPane tab="Core Model" key="2">
                <Spin spinning={loadingModel}>
                  <div>
                    <h3>Core Model</h3>
                    {coreModel ? (
                      <div>
                        <p>
                          <strong>Model Name:</strong> {coreModel.name}
                        </p>
                        <Table
                          columns={fieldColumns}
                          dataSource={coreModel.fields.map((f: Field) => ({ ...f, key: f.key }))}
                          pagination={false}
                          size="small"
                        />
                      </div>
                    ) : (
                      <p>Core model has not been generated yet.</p>
                    )}
                    <Button
                      type="primary"
                      icon={<SyncOutlined />}
                      onClick={handleGenerateCoreModel}
                      loading={loadingModel}
                      style={{ marginTop: '16px' }}
                    >
                      Generate/Update Core Model
                    </Button>
                  </div>
                </Spin>
              </TabPane>
              <TabPane tab="Core View" key="3">
                <Spin spinning={loadingView}>
                  <div>
                    <h3>Core View</h3>
                    {coreView ? (
                      <div>
                        <p>
                          <strong>View Name:</strong> {coreView.name}
                        </p>
                        <Table
                          columns={fieldColumns}
                          dataSource={coreView.fields.map((f: Field) => ({ ...f, key: f.key }))}
                          pagination={false}
                          size="small"
                        />
                      </div>
                    ) : (
                      <p>Core view has not been generated yet.</p>
                    )}
                    <Button
                      type="primary"
                      icon={<SyncOutlined />}
                      onClick={handleGenerateCoreView}
                      loading={loadingView}
                      disabled={!coreModel}
                      style={{ marginTop: '16px' }}
                    >
                      Generate/Update Core View
                    </Button>
                  </div>
                </Spin>
              </TabPane>
              <TabPane tab="Custom Model" key="4">
                <Spin spinning={loadingCustomModel}>
                  <div>
                    <h3>Custom Model</h3>
                    {customModel ? (
                      <div>
                        <p>
                          <strong>Model Name:</strong> {customModel.name}
                        </p>
                        <Table
                          columns={fieldColumns}
                          dataSource={customModel.fields.map((f: Field) => ({ ...f, key: f.key }))}
                          pagination={false}
                          size="small"
                        />
                      </div>
                    ) : (
                      <p>Custom model has not been generated yet.</p>
                    )}
                    <Button
                      type="primary"
                      icon={<SyncOutlined />}
                      onClick={handleGenerateCustomModel}
                      loading={loadingCustomModel}
                      disabled={!coreModel}
                      style={{ marginTop: '16px' }}
                    >
                      Generate/Update Custom Model
                    </Button>
                  </div>
                </Spin>
              </TabPane>
              <TabPane tab="Custom View" key="5">
                <Spin spinning={loadingCustomView}>
                  <div>
                    <h3>Custom View</h3>
                    {customView ? (
                      <div>
                        <p>
                          <strong>View Name:</strong> {customView.name}
                        </p>
                        <Table
                          columns={fieldColumns}
                          dataSource={customView.fields.map((f: Field) => ({ ...f, key: f.key }))}
                          pagination={false}
                          size="small"
                        />
                      </div>
                    ) : (
                      <p>Custom view has not been generated yet.</p>
                    )}
                    <Button
                      type="primary"
                      icon={<SyncOutlined />}
                      onClick={handleGenerateCustomView}
                      loading={loadingCustomView}
                      disabled={!customModel}
                      style={{ marginTop: '16px' }}
                    >
                      Generate/Update Custom View
                    </Button>
                  </div>
                </Spin>
              </TabPane>
            </Tabs>
          </Content>
        </Layout>
      </Modal>

      {/* Semantic Term Selection Modal */}
      <Modal
        title="Add Field - Select Semantic Term"
        visible={showSemanticModal}
        onCancel={() => setShowSemanticModal(false)}
        footer={null}
        width={600}
      >
        <Input
          placeholder="Search semantic terms..."
          value={semanticSearchTerm}
          onChange={(e) => setSemanticSearchTerm(e.target.value)}
          style={{ marginBottom: '16px' }}
        />
        <div className={styles.modalContent}>
          {searchSemanticTerms(semanticTerms, semanticSearchTerm).map((term: EnhancedSemanticTerm) => (
            <div
              key={term.id}
              className={styles.semanticTermCard}
              onClick={() => handleSelectSemanticTerm(term.id)}
            >
              <div className={styles.semanticTermCardName}>
                <strong>{term.node_name}</strong>
                <Button
                  type="primary"
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleSelectSemanticTerm(term.id);
                  }}
                  style={{ float: 'right' }}
                >
                  Add
                </Button>
              </div>
              <div className={styles.semanticTermCardDetails}>
                Technical: {term.technicalName} | Type: {term.dataType}
              </div>
            </div>
          ))}
        </div>
      </Modal>
    </>
  );
}
