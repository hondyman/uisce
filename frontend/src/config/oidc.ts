import { UserManager, WebStorageStateStore, UserManagerSettings } from 'oidc-client-ts';
import { devLog, devError } from '../utils/devLogger';

const issuer = import.meta.env.VITE_OIDC_ISSUER || 'http://100.84.50.65:8089/realms/master';
const clientId = import.meta.env.VITE_OIDC_CLIENT_ID || 'semlayer-frontend';
const redirectUri = import.meta.env.VITE_OIDC_REDIRECT_URI || 'http://localhost:5173/auth/callback';
const postLogoutRedirectUri = import.meta.env.VITE_OIDC_POST_LOGOUT_URI || 'http://localhost:5173/login';

if (!clientId) {
  devError(
    'VITE_OIDC_CLIENT_ID is not set. Authentication will fail. ' +
      'Set it in your .env.local file or environment.',
  );
}

export const oidcSettings: UserManagerSettings = {
  authority: issuer,
  client_id: clientId || '',
  redirect_uri: redirectUri,
  post_logout_redirect_uri: postLogoutRedirectUri,
  response_type: 'code',
  scope: 'openid profile email',
  userStore: new WebStorageStateStore({ store: window.localStorage }),
  stateStore: new WebStorageStateStore({ store: window.localStorage }),
  automaticSilentRenew: false, // managed manually via getValidToken/signinSilent
  loadUserInfo: true,
  monitorSession: false,
};

export const userManager = new UserManager(oidcSettings);

// Log OIDC events in dev to aid debugging.
userManager.events.addUserLoaded((user) => {
  devLog('[OIDC] user loaded', { sub: user.profile?.sub, expired: user.expired });
});

userManager.events.addUserUnloaded(() => {
  devLog('[OIDC] user unloaded');
});

userManager.events.addSilentRenewError((error) => {
  devError('[OIDC] silent renew error', error);
});

export const getOidcIssuer = (): string => issuer;
export const getOidcClientId = (): string => clientId || '';
