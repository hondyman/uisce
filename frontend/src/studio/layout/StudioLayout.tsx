import { EditorPanel } from '../panels/EditorPanel'

interface StudioLayoutProps {
  kernel: any;
}

export function StudioLayout({kernel }: StudioLayoutProps) {
  const panels = kernel.services.plugins.getPanels()

  return (
    <div className="studio-layout">
      <div className="main-content">
        <div className="editor-section">
          <EditorPanel kernel={kernel} />
        </div>

        <div className="panels-section">
          <div className="panels-grid">
            {panels.slice(0, 6).map((panel: any) => (
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