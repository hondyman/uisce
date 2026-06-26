import { render, screen, fireEvent } from '@testing-library/react'
import { vi } from 'vitest'
import { ConfirmProvider, useConfirm } from '../src/components/ConfirmProvider'
import { SnackbarProvider } from 'notistack'

function TestConfirm() {
  const confirm = useConfirm()
  const onClick = async () => {
    const r = await confirm({ title: 'Delete item', description: 'Confirm delete?' })
    (window as any).__confirmResult = r
  }
  return <button onClick={onClick}>Open Confirm</button>
}

describe('ConfirmProvider', () => {
  it('resolves confirm promise with true when confirmed', async () => {
    render(
      <SnackbarProvider>
        <ConfirmProvider>
          <TestConfirm />
        </ConfirmProvider>
      </SnackbarProvider>
    )

    fireEvent.click(screen.getByText('Open Confirm'))
    // dialog should show
    expect(await screen.findByText('Delete item')).toBeInTheDocument()

    // click confirm
    const confirmBtn = screen.getByText('Confirm')
    fireEvent.click(confirmBtn)

    // callback sets __confirmResult
    expect((window as any).__confirmResult).toBe(true)
  })
  it('resolves confirm promise with false when canceled', async () => {
    render(
      <SnackbarProvider>
        <ConfirmProvider>
          <TestConfirm />
        </ConfirmProvider>
      </SnackbarProvider>
    )

    fireEvent.click(screen.getByText('Open Confirm'))
    expect(await screen.findByText('Delete item')).toBeInTheDocument()
    const cancelBtn = screen.getByText('Cancel')
    fireEvent.click(cancelBtn)
    expect((window as any).__confirmResult).toBe(false)
  })
})
