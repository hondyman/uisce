
import TabsManager from './TabsManager';
import './Explorer.css';

/**
 * The main page component for the Explorer feature.
 * It renders the TabsManager which handles the multi-tab workspace.
 * (Drag-and-drop, where needed, is now handled locally via @dnd-kit
 * inside the components that use it.)
 */
export default function ExplorerPage() {
  return (
    <TabsManager />
  );
}
