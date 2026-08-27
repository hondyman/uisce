import { render, screen, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import { DndContext } from '@dnd-kit/core'
import ReportSection from '../../../../src/components/reporting/ReportSection'
import { REPORT_SECTIONS } from '../../../../src/components/reporting/reportingUtils'

const noop = () => {}

function renderSection({
  sectionConfig = {},
  onSectionConfigChange = noop,
  elements = [],
  isLivePreview = false,
}: {
  sectionConfig?: Record<string, any>
  onSectionConfigChange?: (section: string, update: Partial<any>) => void
  elements?: any[]
  isLivePreview?: boolean
} = {}) {
  return render(
    <DndContext>
      <ReportSection
        section={REPORT_SECTIONS.BODY}
        elements={elements}
        onElementUpdate={noop}
        onElementDelete={noop}
        onElementSelect={noop}
        layoutSettings={{
          columns: 1,
          columnSpacing: 24,
          headerTokens: [],
          footerTokens: [],
          pageBreakBetweenRegions: false,
          pageBreakAfterGroup: false,
        }}
        sectionConfig={sectionConfig}
        onSectionConfigChange={onSectionConfigChange}
        isLivePreview={isLivePreview}
        previewData={null}
        availableFieldDefs={[]}
      />
    </DndContext>
  )
}

describe('ReportSection visibility toggling', () => {
  it('renders section header and drop zone in default (visible) state', () => {
    renderSection()
    expect(screen.getByText('Body (Detail)')).toBeInTheDocument()
    expect(screen.getByLabelText(/Drop zone for Body/)).toBeInTheDocument()
  })

  it('does NOT show Hidden chip when section is visible', () => {
    renderSection({ sectionConfig: { [REPORT_SECTIONS.BODY]: { visible: true } } })
    expect(screen.queryByText('Hidden')).not.toBeInTheDocument()
  })

  it('keeps header and drop zone mounted when manually hidden (visible === false)', () => {
    renderSection({
      sectionConfig: { [REPORT_SECTIONS.BODY]: { visible: false } },
    })
    expect(screen.getByText('Body (Detail)')).toBeInTheDocument()
    expect(screen.getByLabelText(/Drop zone for Body/)).toBeInTheDocument()
  })

  it('shows Hidden chip when section is manually hidden', () => {
    renderSection({
      sectionConfig: { [REPORT_SECTIONS.BODY]: { visible: false } },
    })
    expect(screen.getByText('Hidden')).toBeInTheDocument()
  })

  it('clicking the eye icon calls onSectionConfigChange with visible: true when currently hidden', () => {
    const onSectionConfigChange = vi.fn()
    renderSection({
      sectionConfig: { [REPORT_SECTIONS.BODY]: { visible: false } },
      onSectionConfigChange,
    })

    const eyeButton = screen.getByRole('button', { name: /Show section/i })
    fireEvent.click(eyeButton)

    expect(onSectionConfigChange).toHaveBeenCalledWith(REPORT_SECTIONS.BODY, { visible: true })
  })

  it('clicking the eye icon calls onSectionConfigChange with visible: false when currently visible', () => {
    const onSectionConfigChange = vi.fn()
    renderSection({
      sectionConfig: { [REPORT_SECTIONS.BODY]: { visible: true } },
      onSectionConfigChange,
    })

    const eyeButton = screen.getByRole('button', { name: /Hide section/i })
    fireEvent.click(eyeButton)

    expect(onSectionConfigChange).toHaveBeenCalledWith(REPORT_SECTIONS.BODY, { visible: false })
  })

  it('section with visibilityCondition expression is NOT affected by manual hide logic', () => {
    const onSectionConfigChange = vi.fn()
    renderSection({
      sectionConfig: {
        [REPORT_SECTIONS.BODY]: {
          visible: false,
          visibilityCondition: { type: 'operator', operator: '=', left: { type: 'field', field: 'Status' }, right: { type: 'value', value: 'Active' } },
        },
      },
      onSectionConfigChange,
    })
    // Section still renders (expression-driven hiding is handled separately via isHiddenByCondition in live preview)
    expect(screen.getByText('Body (Detail)')).toBeInTheDocument()
    expect(screen.getByText('Dynamic Rule')).toBeInTheDocument()
  })
})
