# Debugging

## Console Noise from Browser Extensions

### MaxListenersExceededWarning

If you see `MaxListenersExceededWarning` in the browser console, this originates from a browser extension (typically MetaMask or another wallet extension) running a liveness probe via `contentscript.js`. **This is not an application error** and requires no action on the application side.

To suppress while debugging: disable wallet/MetaMask extensions in your browser's developer tools Extensions panel, then reload.
