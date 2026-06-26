
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import TabsManager from './TabsManager';
import './Explorer.css';

/**
 * The main page component for the Explorer feature.
 * It renders the TabsManager which handles the multi-tab workspace,
 * and wraps it with the DndProvider for drag-and-drop functionality.
 */
export default function ExplorerPage() {
  return (
    <DndProvider backend={HTML5Backend}>
      <TabsManager />
    </DndProvider>
  );
}