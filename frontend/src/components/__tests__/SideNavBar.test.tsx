import { render, screen } from '@testing-library/react'
import SideNavBar from '../SideNavBar'

describe('SideNavBar', () => {
  it('renders New Workflow button', () => {
    render(<SideNavBar />)
    const btn = screen.getByRole('button', { name: /New Workflow/i })
    expect(btn).toBeInTheDocument()
  })
})
