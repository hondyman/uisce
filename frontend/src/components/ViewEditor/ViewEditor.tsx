// React default import removed — using automatic JSX runtime
import ViewEditorComplete from './ViewEditorComplete';

interface ViewEditorProps {
  viewName: string;
  viewData: any;
  setViewData: (data: any) => void;
  onSave: () => void;
  onValidate: () => void;
  isSaving: boolean;
  isValidating: boolean;
  validationResult: any;
  notification: any;
  setNotification: (notification: any) => void;
  tenantId?: string;
  datasourceId?: string;
}

const ViewEditor: React.FC<ViewEditorProps> = (props) => {
  return <ViewEditorComplete {...props} />;
};

export default ViewEditor;
