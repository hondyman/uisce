import React, { useState, useRef, useEffect } from 'react';
import useBlockableNavigate from '../RouteBlocker/useBlockableNavigate';
import { IconDeviceFloppy, IconSettings, IconUser, IconLogout, IconDatabase, IconChevronDown } from '@tabler/icons-react';
import { useAuth } from '../../contexts/AuthContext';
import IconLockAccess from '@tabler/icons-react/dist/esm/icons/IconLockAccess.mjs';
import { devLog } from '../../utils/devLogger';

interface Props {
  handleSave: () => Promise<void> | void;
  isSaving: boolean;
}

const HeaderActions: React.FC<Props> = ({ handleSave, isSaving }) => {
  const [showSettingsMenu, setShowSettingsMenu] = useState(false);
  const [showFabricMenu, setShowFabricMenu] = useState(false);
  const settingsRef = useRef<HTMLDivElement>(null);
  const fabricRef = useRef<HTMLDivElement>(null);
  const { user, logout } = useAuth();
  const navigate = useBlockableNavigate();

  // Close menus when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (settingsRef.current && !settingsRef.current.contains(event.target as Node)) {
        setShowSettingsMenu(false);
      }
      if (fabricRef.current && !fabricRef.current.contains(event.target as Node)) {
        setShowFabricMenu(false);
      }
    };

    if (showSettingsMenu || showFabricMenu) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [showSettingsMenu, showFabricMenu]);

  const handleLogout = async () => {
  setShowSettingsMenu(false);
  // debug log to ensure logout handler is invoked
  devLog('HeaderActions: logout clicked');
  await logout();
    // Ensure the user is redirected to the login page after logout
    try {
      // allow route blocker to intercept navigation
      void navigate('/login', { replace: true });
    } catch (err) {
      // fallback: reload the page
      window.location.href = '/login';
    }
  };

  return (
    <div className="header-right">
      <div className="header-actions">
        <button onClick={handleSave} className={`btn btn-primary enhanced-save-btn ${isSaving ? 'saving' : ''}`} title="Save semantic model" disabled={isSaving}>
              {isSaving ? (
            <>
              <div className="saving-spinner" />
              <span>Saving...</span>
            </>
          ) : (
            <>
              <IconDeviceFloppy size={18} />
              <span>Save Model</span>
            </>
          )}
        </button>

        {/* Fabric Menu */}
        <div className="fabric-menu-container" ref={fabricRef}>
          <button
            onClick={() => setShowFabricMenu(!showFabricMenu)}
            className="btn btn-outline fabric-btn"
            title="Fabric"
            aria-label="Fabric menu"
          >
            <IconDatabase size={18} />
            <span>Fabric</span>
            <IconChevronDown size={14} />
          </button>

          {showFabricMenu && (
            <div className="fabric-dropdown">
              <button
                  onClick={() => {
                  setShowFabricMenu(false);
                  void navigate('/fabric/preaggregations');
                }}
                className="fabric-menu-item"
                title="Pre-Aggregations"
              >
                <IconDatabase size={16} />
                <span>Pre-Aggregations</span>
              </button>
              <button
                onClick={() => {
                  setShowFabricMenu(false);
                  void navigate('/fabric/calculations');
                }}
                className="fabric-menu-item"
                title="Calculations Library"
              >
                <IconDatabase size={16} />
                <span>Calculations Library</span>
              </button>
              <button
                onClick={() => {
                  setShowFabricMenu(false);
                  void navigate('/conversational-query');
                }}
                className="fabric-menu-item"
                title="Conversational Query"
              >
                <IconDatabase size={16} />
                <span>Conversational Query</span>
              </button>
            </div>
          )}
        </div>

  {/* settings menu trigger (contains Sign Out as menu item) */}

        {/* Settings Menu */}
        <div className="settings-menu-container" ref={settingsRef}>
            <button
            onClick={() => setShowSettingsMenu(!showSettingsMenu)}
            className="btn btn-outline settings-btn"
            title="Settings"
            aria-label="Settings menu"
          >
            <IconSettings size={18} />
          </button>

          {showSettingsMenu && (
            <div className={`settings-dropdown debug-outline`}>
              <div className="settings-user-info">
                <div className="user-avatar">
                  <IconUser size={16} />
                </div>
                <div className="user-details">
                  <div className="user-name">{user?.name || 'User'}</div>
                  <div className="user-email">{user?.email}</div>
                </div>
              </div>

              <div className="settings-divider" />

              {/* Move IP Whitelist above Sign Out and add simple divider for clarity */}
              <button
                  onClick={() => {
                  devLog('IP Whitelist clicked');
                  setShowSettingsMenu(false);
                  void navigate('/fabric/ip-whitelist');
                }}
                className="settings-menu-item ip-whitelist-item"
                data-testid="ip-whitelist-item"
                title="IP Whitelist"
              >
                <IconLockAccess size={16} />
                <span>IP Whitelist</span>
              </button>
              <div className="settings-divider" />
              <button
                onClick={handleLogout}
                className="settings-menu-item logout-item"
                title="Sign out"
              >
                <IconLogout size={16} />
                <span>Sign Out</span>
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default HeaderActions;
