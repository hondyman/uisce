interface CodeBlockProps {
  language: string;
  content: string;
}

export default function CodeBlock({ language, content }: CodeBlockProps) {
  // In a real app, you'd use a syntax highlighting library like Prism.js or highlight.js
  return (
    <pre className={`code-block language-${language}`}>{content}</pre>
  );
}