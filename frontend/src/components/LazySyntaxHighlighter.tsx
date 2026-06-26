import { lazy, FC, createElement } from 'react';

// Lazily load react-syntax-highlighter and the dracula style.
// Returns a default export component that renders the Prism highlighter with the dracula style.
const LazySyntaxHighlighter = lazy(async () => {
  const mod = await import('react-syntax-highlighter');
  const styleMod = await import('react-syntax-highlighter/dist/esm/styles/prism');
  // Prefer named Prism import, fall back to module shape
  const Prism = (mod as any).Prism || (mod as any).default?.Prism || (mod as any).default || (mod as any);
  const dracula = (styleMod as any).dracula || (styleMod as any).default?.dracula;

  const Comp: FC<any> = (props) => {
    return createElement(Prism, { style: dracula, ...props }, props.children);
  };

  return { default: Comp } as any;
});

export default LazySyntaxHighlighter;
