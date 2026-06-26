import { FC } from 'react';
import { Handle, Position } from 'reactflow';

// styles moved to App.css as .schema-container, .table-container and .container-label

export const SchemaContainerNode: FC<{ data: { label?: string } }> = ({ data }) => {
  return (
    <div className="schema-container header-schema">
      <div className="container-label">{data.label}</div>
    </div>
  );
};

export const TableContainerNode: FC<{ data: { label?: string } }> = ({ data }) => {
  return (
    <div className="table-container header-table">
      <div className="container-label">{data.label}</div>
    </div>
  );
};

export const ColumnNode: FC<{ data: { label?: string; isPrimary?: boolean } }> = ({ data }) => {
  const classes = ['column-node'];
  if (data.isPrimary) classes.push('column-node--center');

  return (
    <div className={classes.join(' ')}>
      <Handle type="target" position={Position.Top} />
      <div className="column-content">{data.label}</div>
      <Handle type="source" position={Position.Bottom} />
    </div>
  );
};
