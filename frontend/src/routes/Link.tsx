import React from 'react';
import {
  Link as RRLink,
  NavLink as RRNavLink,
  LinkProps,
  NavLinkProps,
} from 'react-router-dom';
import { useLocale } from '../i18n/useLocale';
import { localePath, Locale } from '../i18n/locales';

function localePrefixed(locale: Locale, to: LinkProps['to']): LinkProps['to'] {
  if (typeof to !== 'string') return to;
  if (!to.startsWith('/')) return to;
  return localePath(locale, to);
}

export function Link({ to, ...rest }: LinkProps) {
  const locale = useLocale();
  return <RRLink to={localePrefixed(locale, to)} {...rest} />;
}

export function NavLink({ to, ...rest }: NavLinkProps) {
  const locale = useLocale();
  return <RRNavLink to={localePrefixed(locale, to)} {...rest} />;
}
