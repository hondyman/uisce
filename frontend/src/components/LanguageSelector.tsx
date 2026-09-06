import React, { useState } from 'react';
import { IconButton, Menu, MenuItem, ListItemText, ListItemIcon } from '@mui/material';
import TranslateIcon from '@mui/icons-material/Translate';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';
import { useLocale } from '../i18n/useLocale';
import { stripLocale } from '../i18n/locales';
import useUserAPI from '../hooks/useUserAPI';
import { useAuth } from '../contexts/AuthContext';

const LANGUAGES = [
  { code: 'en', label: 'English' },
  { code: 'es', label: 'Español' },
  { code: 'fr', label: 'Français' },
  { code: 'xx', label: 'Test' },
];

const LanguageSelector: React.FC = () => {
  const { i18n } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const locale = useLocale();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const handleOpen = (ev: React.MouseEvent<HTMLElement>) => setAnchorEl(ev.currentTarget);
  const handleClose = () => setAnchorEl(null);

  const { getUserPreferences, updateUserPreferences } = useUserAPI();
  const auth = useAuth();

  const handleChangeLanguage = async (code: string) => {
    await i18n.changeLanguage(code);
    const currentPath = stripLocale(location.pathname);
    navigate(`/${code}${currentPath}`);
    try { localStorage.setItem('appLocale', code); } catch (e) { /* ignore */ }
    try {
      if (auth?.user?.id) {
        await updateUserPreferences(auth.user.id, { language: code });
      }
    } catch (err) {
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
