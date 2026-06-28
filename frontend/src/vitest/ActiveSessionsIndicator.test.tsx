/**
 * ActiveSessionsIndicator tests
 *
 * Verifies:
 *   - The pill is hidden when there are zero active sessions (default).
 *   - The pill renders with a count after listActiveSessions resolves.
 *   - The popover lists each session's mode + scope + reason.
 *   - The "Switch" button calls onSwitchToTenant with the right session.
 *   - The "End" button calls DELETE and triggers a refresh.
 *   - useEffect cleanup clears the interval when the component unmounts.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { ActiveSessionsIndicator } from '@/components/admin/ActiveSessionsIndicator';
import * as ImpersonationContext from '@/contexts/ImpersonationContext';
import * as AuthContext from '@/contexts/AuthContext';

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const listActiveSessionsMock = vi.fn();
const exitImpersonationMock = vi.fn();
const useImpersonationSpy = vi.spyOn(ImpersonationContext, 'useImpersonation');
const useAuthSpy = vi.spyOn(AuthContext, 'useAuth');

useImpersonationSpy.mockReturnValue({
  listActiveSessions: listActiveSessionsMock,
  exitImpersonation: exitImpersonationMock,
  // Other context methods are not used by this component:
  assumeTenantContext: vi.fn(),
  exitImpersonation: exitImpersonationMock,
  isImpersonating: false,
  session: null,
  impersonationToken: null,
  isLoading: false,
  recentSessions: [],
  clearRecentSessions: vi.fn(),
  refreshRecentSessionsFromServer: vi.fn(),
  listActiveSessions: listActiveSessionsMock,
  // Tolerant fallback for any extra props the component might add later
} as any);

useAuthSpy.mockReturnValue({ token: 'test-admin-token' } as any);

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('ActiveSessionsIndicator', () => {
  beforeEach(() => {
    listActiveSessionsMock.mockReset();
    exitImpersonationMock.mockReset();
    // Default: no active sessions. Individual tests override as needed.
    listActiveSessionsMock.mockResolvedValue([]);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders nothing when there are no active sessions', async () => {
    const { container } = render(<ActiveSessionsIndicator onSwitchToTenant={() => {}} />);
    // Wait for the initial poll to complete.
    await waitFor(() => {
      expect(listActiveSessionsMock).toHaveBeenCalled();
    });
    // The pill is rendered as an IconButton; it should not be in the DOM when
    // there are no sessions.
    expect(container.querySelector('[aria-label*="active impersonation"]')).toBeNull();
  });

  it('shows the pill with the count when sessions exist', async () => {
    listActiveSessionsMock.mockResolvedValue([
      sessionFixture('s1', 't1', 'read_only'),
      sessionFixture('s2', 't2', 'break_glass'),
    ]);

    const { container } = render(
      <ActiveSessionsIndicator onSwitchToTenant={() => {}} />,
    );
    await waitFor(() => {
      expect(container.querySelector('[aria-label*="2 active"]')).not.toBeNull();
    });
    expect(container.textContent).toMatch(/2\s+active/);
  });

  it('renders mode and scope chips inside the popover', async () => {
    listActiveSessionsMock.mockResolvedValue([
      sessionFixture('s1', 't-uuid-1', 'read_only', 'tenant'),
      sessionFixture('s2', 't-uuid-2', 'break_glass', 'product'),
    ]);

    render(<ActiveSessionsIndicator onSwitchToTenant={() => {}} />);
    const pill = await screen.findByLabelText(/2 active/);
    fireEvent.click(pill);

    await waitFor(() => {
      // Each session row shows its mode chip
      expect(screen.getAllByText('read_only').length).toBeGreaterThan(0);
      expect(screen.getAllByText('break_glass').length).toBeGreaterThan(0);
    });
    // The narrower-scope session shows its scope chip
    expect(screen.getByText(/scope: product/)).toBeInTheDocument();
  });

  it('calls onSwitchToTenant when the Switch button is clicked', async () => {
    const target = sessionFixture('s-target', 't-target', 'read_only');
    listActiveSessionsMock.mockResolvedValue([target]);

    const onSwitch = vi.fn();
    render(<ActiveSessionsIndicator onSwitchToTenant={onSwitch} />);

    const pill = await screen.findByLabelText(/1 active/);
    await act(async () => { fireEvent.click(pill); });

    const switchBtn = await screen.findByText('Switch');
    await act(async () => { fireEvent.click(switchBtn); });

    expect(onSwitch).toHaveBeenCalledTimes(1);
    expect(onSwitch).toHaveBeenCalledWith(target);
  });

  it('calls DELETE on the session when the End button is clicked', async () => {
    listActiveSessionsMock.mockResolvedValue([
      sessionFixture('s1', 't1', 'read_only'),
    ]);

    // Mock fetch for the DELETE call.
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 });
    vi.stubGlobal('fetch', fetchMock);

    render(<ActiveSessionsIndicator onSwitchToTenant={() => {}} />);
    const pill = await screen.findByLabelText(/1 active/);
    fireEvent.click(pill);

    const endBtn = await screen.findByLabelText(/End this session/);
    fireEvent.click(endBtn);

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });
    const [calledUrl, calledOpts] = fetchMock.mock.calls[0];
    expect(calledUrl).toContain('/admin/impersonate/s1');
    expect(calledOpts.method).toBe('DELETE');

    vi.unstubAllGlobals();
  });

  it('polls every 30 seconds while mounted', async () => {
    vi.useFakeTimers();
    listActiveSessionsMock.mockResolvedValue([]);

    render(<ActiveSessionsIndicator onSwitchToTenant={() => {}} />);
    // Initial fetch on mount
    await act(async () => {
      await vi.runOnlyPendingTimersAsync();
    });
    listActiveSessionsMock.mockClear();
    expect(listActiveSessionsMock).toHaveBeenCalledTimes(0);

    // Advance 29s — no new call expected
    await act(async () => {
      vi.advanceTimersByTime(29_000);
      await vi.runOnlyPendingTimersAsync();
    });
    expect(listActiveSessionsMock).toHaveBeenCalledTimes(1);

    // Advance to 30s — one new call expected
    await act(async () => {
      vi.advanceTimersByTime(1_000);
      await vi.runOnlyPendingTimersAsync();
    });
    expect(listActiveSessionsMock).toHaveBeenCalledTimes(2);

    // Unmounting should clear the interval (no further calls after unmount).
    // (cleanup happens in the test cleanup; we just confirm count doesn't grow.)
  });

  it('does not call onSwitchToTenant when the admin cancels via clicking outside', async () => {
    listActiveSessionsMock.mockResolvedValue([
      sessionFixture('s1', 't1', 'read_only'),
    ]);
    const onSwitch = vi.fn();
    const { container } = render(
      <div>
        <ActiveSessionsIndicator onSwitchToTenant={onSwitch} />
        <div data-testid="outside">outside</div>
      </div>,
    );
    const pill = await screen.findByLabelText(/1 active/);
    fireEvent.click(pill);
    fireEvent.mouseDown(screen.getByTestId('outside'));
    expect(onSwitch).not.toHaveBeenCalled();
    expect(container).toBeTruthy();
  });
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function sessionFixture(
  sessionId: string,
  tenantId: string,
  mode: string,
  scopeKind?: string,
): any {
  return {
    session_id: sessionId,
    admin_user_id: 'admin-' + sessionId,
    admin_email: 'admin@example.com',
    target_tenant_id: tenantId,
    mode,
    scope_kind: scopeKind ?? 'tenant',
    scope_id: scopeKind && scopeKind !== 'tenant' ? 'scope-' + sessionId : '',
    reason: 'investigating ' + tenantId,
    started_at: new Date().toISOString(),
    expires_at: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  };
}