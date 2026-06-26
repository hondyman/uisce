import React, { useState, useMemo, useEffect } from 'react';
import { useNotification } from '../../hooks/useNotification';
import {
  DndContext,
  DragEndEvent,
  useDroppable,
  useDraggable,
} from '@dnd-kit/core';
import { useMutation, useQuery as _useQuery, gql } from '@apollo/client';
import './ScreenBuilder.css';
import { devError } from '../../utils/devLogger';
import ActionButton from '../../components/ui/ActionButton';

// ============================================================================
// GRAPHQL OPERATIONS
// ============================================================================

const CREATE_SCREEN = gql`
  mutation CreateScreen(
    $tenantId: uuid!
    $boType: String!
    $screenName: String!
    $screenType: String!
    $fields: jsonb!
    $filters: jsonb!
    $actions: jsonb!
    $permissions: jsonb!
  ) {
    insert_screen_configs_one(
      object: {
        tenant_id: $tenantId
        bo_type: $boType
        screen_name: $screenName
        screen_type: $screenType
        layout_json: $fields
        filters_json: $filters
        actions_json: $actions
        permissions_json: $permissions
        is_published: false
        created_by: "user-id"
      }
    ) {
      id
      screen_name
      created_at
    }
  }
`;

const UPDATE_SCREEN = gql`
  mutation UpdateScreen(
    $tenantId: uuid!
    $screenId: uuid!
    $fields: jsonb!
  ) {
    update_screen_configs(
      where: { id: { _eq: $screenId } }
      _set: { layout_json: $fields }
    ) {
      affected_rows
    }
  }
`;

const PUBLISH_SCREEN = gql`
  mutation PublishScreen($screenId: uuid!) {
    update_screen_configs(
      where: { id: { _eq: $screenId } }
      _set: { is_published: true }
    ) {
      affected_rows
    }
  }
`;

// ============================================================================
// TYPES
// ============================================================================

interface ScreenField {
  field: string;
  label: string;
  type: 'text' | 'number' | 'date' | 'select' | 'textarea';
  order: number;
  required?: boolean;
  searchable?: boolean;
  editable?: boolean;
}

interface ScreenBuilderProps {
  tenantId: string;
  boType: string;
  onScreenCreated?: (screenId: string) => void;
}

type ScreenType = 'detail' | 'list' | 'create' | 'edit';

interface DragData {
  type: string;
  label: string;
}

// ============================================================================
// DRAGGABLE FIELD COMPONENT
// ============================================================================

interface DraggableFieldProps {
  id: string;
  label: string;
}

const DraggableField: React.FC<DraggableFieldProps> = ({ id, label }) => {
  const { attributes, listeners, setNodeRef, transform, isDragging } =
    useDraggable({
      id,
      data: { type: 'field', label },
    });

  return (
    <div
      ref={setNodeRef}
      {...listeners}
      {...attributes}
      className={`draggable-field ${isDragging ? 'dragging' : ''}`}
      style={
        transform
          ? ((): React.CSSProperties => {
              const cssVars: Record<string, string> = {
                '--translate-x': `${transform.x}px`,
                '--translate-y': `${transform.y}px`,
              };
              return cssVars as unknown as React.CSSProperties;
            })()
          : undefined
      }
    >
      📦 {label}
    </div>
  );
};

// ============================================================================
// DROPPABLE PREVIEW COMPONENT
// ============================================================================

interface DroppablePreviewProps {
  fields: ScreenField[];
  onFieldsChange: (fields: ScreenField[]) => void;
  onRemoveField: (order: number) => void;
}

const DroppablePreview: React.FC<DroppablePreviewProps> = ({
  fields,
  onFieldsChange: _onFieldsChange,
  onRemoveField,
}) => {
  const { setNodeRef } = useDroppable({
    id: 'preview',
  });

  return (
    <div ref={setNodeRef} className="droppable-preview">
      <h3>Screen Preview</h3>
      {fields.length === 0 ? (
        <p className="empty-state">Drag fields here to build your screen</p>
      ) : (
        <div className="preview-fields">
          {fields
            .sort((a, b) => a.order - b.order)
            .map((field) => (
              <div key={field.order} className="preview-field">
                <div className="field-label">{field.label}</div>
                <div className="field-input">
                  {field.type === 'textarea' ? (
                    <textarea disabled placeholder={field.label} />
                  ) : field.type === 'date' ? (
                    <input type="date" disabled placeholder={field.label} />
                  ) : field.type === 'number' ? (
                    <input type="number" disabled placeholder={field.label} />
                  ) : (
                    <input type="text" disabled placeholder={field.label} />
                  )}
                </div>
                <button
                  className="remove-btn"
                  onClick={() => onRemoveField(field.order)}
                >
                  ✕
                </button>
              </div>
            ))}
        </div>
      )}
    </div>
  );
};

// ============================================================================
// MAIN SCREEN BUILDER COMPONENT
// ============================================================================

export const ScreenBuilder: React.FC<ScreenBuilderProps> = ({
  tenantId,
  boType,
  onScreenCreated,
}) => {
  const [screenName, setScreenName] = useState('');
  const [screenType, setScreenType] = useState<ScreenType>('detail');
  const [fields, setFields] = useState<ScreenField[]>([]);
  const [filterFields, setFilterFields] = useState<ScreenField[]>([]);
  const [actions, setActions] = useState<string[]>(['save', 'delete', 'cancel']);
  const [nextOrder, setNextOrder] = useState(1);

  const [createScreen] = useMutation(CREATE_SCREEN);
  const [_updateScreen] = useMutation(UPDATE_SCREEN);
  const [_publishScreen] = useMutation(PUBLISH_SCREEN);

  // Available fields for drag-drop
  const availableFields: DraggableFieldProps[] = [
    { id: 'field_name', label: 'Name' },
    { id: 'field_address', label: 'Address' },
    { id: 'field_city', label: 'City' },
    { id: 'field_country', label: 'Country' },
    { id: 'field_phone', label: 'Phone' },
    { id: 'field_email', label: 'Email' },
    { id: 'field_status', label: 'Status' },
    { id: 'field_created_date', label: 'Created Date' },
    { id: 'field_notes', label: 'Notes' },
  ];

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (!over || over.id !== 'preview') {
      return;
    }

  const fieldData = active.data?.current as DragData | undefined;
  if (!fieldData) return;

    const newField: ScreenField = {
      field: active.id as string,
      label: fieldData.label,
      type: 'text',
      order: nextOrder,
      required: false,
      searchable: false,
      editable: true,
    };

    setFields([...fields, newField]);
    setNextOrder(nextOrder + 1);
  };

  const handleRemoveField = (order: number) => {
    setFields(fields.filter((f) => f.order !== order));
  };

  const handleSaveScreen = async () => {
    const notification = useNotification();
    if (!screenName.trim()) {
      notification.error('Please enter a screen name');
      return;
    }

    try {
      const result = await createScreen({
        variables: {
          tenantId,
          boType,
          screenName,
          screenType,
          fields: fields.map((f) => ({
            field: f.field,
            label: f.label,
            type: f.type,
            order: f.order,
            required: f.required,
            searchable: f.searchable,
            editable: f.editable,
          })),
          filters: filterFields.map((f) => ({
            field: f.field,
            label: f.label,
            type: f.type,
            order: f.order,
          })),
          actions,
          permissions: {
            admin: ['save', 'delete', 'view'],
            user: ['view', 'save'],
          },
        },
      });

      if (result.data?.insert_screen_configs_one?.id) {
        notification.success(`Screen "${screenName}" created successfully!`);
        if (onScreenCreated) {
          onScreenCreated(result.data.insert_screen_configs_one.id);
        }
        // Reset form
        setScreenName('');
        setFields([]);
        setFilterFields([]);
        setNextOrder(1);
      }
    } catch (error) {
      devError('Error creating screen:', error);
      notification.error(`Failed to create screen: ${error}`);
    }
  };

  return (
    <DndContext onDragEnd={handleDragEnd}>
      <div className="screen-builder">
        <div className="screen-builder-header">
          <h2>🎨 Workday-Style Screen Builder</h2>
          <p>Drag fields below to design your screen in seconds!</p>
        </div>

        <div className="screen-builder-container">
          {/* Configuration Panel */}
          <div className="config-panel">
            <h3>Screen Configuration</h3>

            <div className="form-group">
              <label>Screen Name</label>
              <input
                type="text"
                value={screenName}
                onChange={(e) => setScreenName(e.target.value)}
                placeholder="e.g., Customer Details"
              />
            </div>

            <div className="form-group">
              <label htmlFor="screenTypeSelect">Screen Type</label>
                <select
                id="screenTypeSelect"
                title="Select the type of screen"
                value={screenType}
                onChange={(e) => setScreenType(e.target.value as ScreenType)}
              >
                <option value="detail">Detail View</option>
                <option value="list">List View</option>
                <option value="create">Create Form</option>
                <option value="edit">Edit Form</option>
              </select>
            </div>

            <div className="form-group">
              <label>Actions</label>
              <div className="actions-list">
                {['save', 'delete', 'cancel', 'approve', 'reject'].map((action) => (
                  <label key={action} className="checkbox">
                    <input
                      type="checkbox"
                      checked={actions.includes(action)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setActions([...actions, action]);
                        } else {
                          setActions(actions.filter((a) => a !== action));
                        }
                      }}
                    />
                    {action}
                  </label>
                ))}
              </div>
            </div>

            <div className="form-actions">
              <ActionButton variant="primary" onClick={handleSaveScreen}>💾 Save Screen</ActionButton>
              <ActionButton variant="secondary" onClick={() => setFields([])}>🔄 Reset</ActionButton>
            </div>
          </div>

          {/* Field Palette */}
          <div className="field-palette">
            <h3>Available Fields</h3>
            <div className="fields-list">
              {availableFields.map((field) => (
                <DraggableField key={field.id} id={field.id} label={field.label} />
              ))}
            </div>
          </div>

          {/* Screen Preview */}
          <DroppablePreview
            fields={fields}
            onFieldsChange={setFields}
            onRemoveField={handleRemoveField}
          />
        </div>

        {/* Field Count */}
        <div className="field-count">
          <strong>Fields Added:</strong> {fields.length}
        </div>
      </div>
    </DndContext>
  );
};

export default ScreenBuilder;
