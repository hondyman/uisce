// React default import removed — file uses only JSX and no React namespace types
import ColumnActionsPanel from './ColumnActionsPanel';
import { SelectedColumn } from './types';

interface ActionsPanelProps {
  selectedColumn: SelectedColumn | null;
  addDimension: (..._args: any[]) => any;
  addMeasure: (..._args: any[]) => any;
  addFilter: (..._args: any[]) => any;
  getBusinessTermForColumn: (_nodeId: string, _columnName: string) => any;
}

const ActionsPanel: React.FC<ActionsPanelProps> = ({
  selectedColumn,
  addDimension,
  addMeasure,
  addFilter,
  getBusinessTermForColumn,
}) => {
  if (!selectedColumn) return null;

  return (
    <section className="actions-panel">
      <ColumnActionsPanel
        selectedColumn={selectedColumn}
        addDimension={addDimension}
        addMeasure={addMeasure}
        addFilter={addFilter}
        getBusinessTermForColumn={getBusinessTermForColumn}
      />
    </section>
  );
};

export default ActionsPanel;
