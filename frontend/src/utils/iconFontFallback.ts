// Lightweight runtime fallback for Material Symbols font glyphs.
// If the Material Symbols font doesn't load, replace known text glyphs
// (e.g. "more_horiz", "grid_view") with inline SVG fallbacks.

const ICON_SVGS: Record<string, string> = {
  more_horiz: `
    <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="6" cy="12" r="1.5" />
      <circle cx="12" cy="12" r="1.5" />
      <circle cx="18" cy="12" r="1.5" />
    </svg>
  `,
  grid_view: `
    <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <rect x="3" y="3" width="8" height="8" />
      <rect x="13" y="3" width="8" height="8" />
      <rect x="3" y="13" width="8" height="8" />
      <rect x="13" y="13" width="8" height="8" />
    </svg>
  `,
  schema: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="12" cy="6" r="2" />
      <circle cx="6" cy="18" r="2" />
      <circle cx="18" cy="18" r="2" />
      <path d="M12 8v6" />
      <path d="M12 14l4 4" />
      <path d="M12 14l-4 4" />
    </svg>
  `,
  ios_share: `
    <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M12 2L12 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
      <path d="M5 9l7-7 7 7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
      <rect x="4" y="15" width="16" height="6" rx="2" />
    </svg>
  `,
  person: `
    <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="12" cy="8" r="3" />
      <path d="M5 20c1.5-4 5.5-6 7-6s5.5 2 7 6" />
    </svg>
  `,
  school: `
    <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M12 2L2 7l10 5 10-5-10-5z" />
      <path d="M2 17l10 5 10-5" />
      <path d="M2 12l10 5 10-5" />
    </svg>
  `,
  corporate_fare: `
    <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M3 13h18v8H3z" />
      <path d="M7 13V6l5-3 5 3v7" fill="none" stroke="currentColor" stroke-width="0" />
    </svg>
  `,
  verified: `
    <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M12 2l1.9 3.9L18 7l-2.1 2.1L16 15l-4-2-4 2 .1-5.9L4 7l4.1-.1L12 2z" />
    </svg>
  `,
  search: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="11" cy="11" r="6"></circle>
      <path d="M21 21l-4.35-4.35"></path>
    </svg>
  `,
  close: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M18 6L6 18"></path>
      <path d="M6 6l12 12"></path>
    </svg>
  `,
  arrow_forward: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M5 12h14"></path>
      <path d="M12 5l7 7-7 7"></path>
    </svg>
  `,
};

function replaceIcons() {
  try {
    const els = Array.from(document.querySelectorAll<HTMLElement>('.material-symbols-outlined'));
    els.forEach((el) => {
      if (el.dataset.icon) return; // already replaced
      const name = (el.textContent || '').trim();
      if (!name) return;
      const key = name;
      const svg = ICON_SVGS[key];
      if (svg) {
        el.innerHTML = svg;
        el.setAttribute('data-icon', key);
        el.setAttribute('role', 'img');
        el.style.display = 'inline-flex';
        el.style.alignItems = 'center';
        el.style.justifyContent = 'center';
      }
    });
  } catch (err) {
    // ignore
  }
}

export function initIconFontFallback(timeoutMs = 1500) {
  if (typeof document === 'undefined') return;
  const fontName = "Material Symbols Outlined";

  // If Font Loading API is available, try to load/check the font.
  if ((document as any).fonts && typeof (document as any).fonts.check === 'function') {
    try {
      const check = (document as any).fonts.check(`1rem "${fontName}"`);
      if (check) return; // font available

      // Attempt to load, then wait up to timeout
      (document as any).fonts.load(`1rem "${fontName}"`).then(() => {
        const ok = (document as any).fonts.check(`1rem "${fontName}"`);
        if (!ok) {
          replaceIcons();
        }
      }).catch(() => {
        // loading failed
        replaceIcons();
      });

      // Fallback: if not loaded after timeout, replace
      setTimeout(() => {
        const ok2 = (document as any).fonts.check(`1rem "${fontName}"`);
        if (!ok2) replaceIcons();
      }, timeoutMs);
    } catch (err) {
      replaceIcons();
    }
    return;
  }

  // No Font Loading API: fallback after timeout
  setTimeout(replaceIcons, timeoutMs);
}

// Auto-init on import for convenience
if (typeof window !== 'undefined') {
  // run after a small delay so DOM exists
  window.addEventListener('load', () => initIconFontFallback(1500));
}

export default initIconFontFallback;
