// React default import not required with new JSX transform
import { ControlButton } from 'reactflow';
import { Map as MapIcon } from '@mui/icons-material';

interface ErdControlsProps {
  onToggleMinimap: () => void;
}

const ErdControls: React.FC<ErdControlsProps> = ({ onToggleMinimap }) => {
  return (
    <div className="react-flow__controls">
      <ControlButton onClick={onToggleMinimap} title="Toggle Minimap">
        <MapIcon />
      </ControlButton>
    </div>
  );
};

export default ErdControls;
