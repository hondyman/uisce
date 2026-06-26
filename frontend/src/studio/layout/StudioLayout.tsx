import { EditorPanel } from '../panels/EditorPanel'

export function StudioLayout({ kernel }) {
  const panels = kernel.services.plugins.getPanels()

  return (
    <div className="studio-layout">
      <div className="main-content">
        <div className="editor-section">
          <EditorPanel kernel={kernel} />
        </div>

        <div className="panels-section">
          <div className="panels-grid">
            {panels.slice(0, 6).map(panel => (
              <div key={panel.id} className="panel-container">
                <panel.component kernel={kernel} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}