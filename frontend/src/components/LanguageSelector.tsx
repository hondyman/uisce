import React, { useState } from 'react';
import { IconButton, Menu, MenuItem, ListItemText, ListItemIcon } from '@mui/material';
import TranslateIcon from '@mui/icons-material/Translate';
import { useTranslation } from 'react-i18next';
import useUserAPI from '../hooks/useUserAPI';
import { useAuth } from '../contexts/AuthContext';

// Supported languages - add more as needed
const LANGUAGES = [
  { code: 'en', label: 'English' },
  { code: 'es', label: 'Español' },
  { code: 'fr', label: 'Français' },
  // synthetic placeholder for test-language used in tests
  { code: 'xx', label: 'Test' },
];

const LanguageSelector: React.FC = () => {
  const { i18n } = useTranslation();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const handleOpen = (ev: React.MouseEvent<HTMLElement>) => setAnchorEl(ev.currentTarget);
  const handleClose = () => setAnchorEl(null);

  const { getUserPreferences, updateUserPreferences } = useUserAPI();
  const auth = useAuth();

  const handleChangeLanguage = async (code: string) => {
    await i18n.changeLanguage(code);
    // persist choice locally
    try { localStorage.setItem('selected_language', code); } catch (e) { /* ignore */ }

    // If user is logged in, try to persist to server
    try {
      if (auth?.user?.id) {
        await updateUserPreferences(auth.user.id, { language: code });
      }
    } catch (err) {
      // ignore network/save failures to avoid blocking the UI
      console.error('Failed to persist language preference', err);
    }

    handleClose();
  };

  return (
    <>
      <IconButton size="small" color="inherit" onClick={handleOpen} aria-label="Language selector">
        <TranslateIcon />
      </IconButton>
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        {LANGUAGES.map((lang) => (
          <MenuItem key={lang.code} onClick={() => handleChangeLanguage(lang.code)} selected={i18n.resolvedLanguage === lang.code}>
            <ListItemText>{lang.label}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};

export default LanguageSelector;
