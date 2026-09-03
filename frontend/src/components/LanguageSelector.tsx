import React, { useState } from 'react';
import { IconButton, Menu, MenuItem, ListItemText } from '@mui/material';
import TranslateIcon from '@mui/icons-material/Translate';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { LOCALES, NATIVE_NAMES, Locale, localePath } from '../i18n/locales';
import { useLocale } from '../i18n/useLocale';
import useUserAPI from '../hooks/useUserAPI';
import { useAuth } from '../contexts/AuthContext';

const LanguageSelector: React.FC = () => {
  const { i18n, t } = useTranslation();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const current = useLocale();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const { updateUserPreferences } = useUserAPI();
  const auth = useAuth();

  const handleOpen = (ev: React.MouseEvent<HTMLElement>) => setAnchorEl(ev.currentTarget);
  const handleClose = () => setAnchorEl(null);

  const handleChangeLanguage = async (code: Locale) => {
    const rest = pathname.replace(/^\/[^/]+/, '') || '/';
    const search = window.location.search;
    const hash = window.location.hash;
    handleClose();

    // If the current path is un-prefixed (e.g. /login, /api-studio), don't
    // navigate — there's no locale segment to rewrite. Just change language
    // and persist; the locale will apply to the next navigation.
    if (!pathname.split('/')[1] || !LOCALES.includes(pathname.split('/')[1] as Locale)) {
      try {
        await i18n.changeLanguage(code);
      } catch (err) {
        console.error('Failed to change language', err);
      }
    } else {
      navigate(`${localePath(code, rest)}${search}${hash}`);
    }

    try { localStorage.setItem('appLocale', code); } catch { /* ignore */ }

    try {
      if (auth?.user?.id) {
        await updateUserPreferences(auth.user.id, { language: code });
      }
    } catch (err) {
      console.error('Failed to persist language preference', err);
    }
  };

  return (
    <>
      <IconButton
        size="small"
        color="inherit"
        onClick={handleOpen}
        aria-label={t('nav.language')}
        title={t('nav.language')}
      >
        <TranslateIcon />
      </IconButton>
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleClose}
      >
        {LOCALES.map((lang) => (
          <MenuItem
            key={lang}
            onClick={() => handleChangeLanguage(lang)}
            selected={i18n.resolvedLanguage === lang || current === lang}
            lang={lang}
          >
            <ListItemText>{NATIVE_NAMES[lang]}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
    </>
  );
};

export default LanguageSelector;
