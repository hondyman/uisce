import React, { useState, useEffect, useRef } from 'react';
import { Box, TextField, IconButton, Tooltip, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import SearchIcon from '@mui/icons-material/Search';
import TextFieldsIcon from '@mui/icons-material/TextFields';
import RegexIcon from '@mui/icons-material/Percent';
import TitleIcon from '@mui/icons-material/Title';

interface CodeSearchProps {
  editor: any | null;
  monaco: any | null;
}

const escapeRegExp = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

const CodeSearch: React.FC<CodeSearchProps> = ({ editor, monaco }) => {
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState(query);
  const [decorations, setDecorations] = useState<string[]>([]);
  const [matches, setMatches] = useState<any[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [options, setOptions] = useState<{ matchCase: boolean; isRegex: boolean; wholeWord: boolean }>({ matchCase: false, isRegex: false, wholeWord: false });
  const [regexError, setRegexError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  // debounce query for performance
  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedQuery(query), 250);
    return () => window.clearTimeout(t);
  }, [query]);

  useEffect(() => {
    if (!editor || !monaco) return;
    // when debouncedQuery changes, find matches and apply decorations
    const q = debouncedQuery;
    if (!q) {
      setMatches([]);
      setCurrentIndex(0);
      setDecorations((prev) => {
        try { return editor.deltaDecorations(prev, []); } catch { return []; }
      });
      setRegexError(null);
      return;
    }

    const model = editor.getModel();
    if (!model) return;

    // build search string and regex flags based on options
    const { matchCase, isRegex, wholeWord } = options;
    let searchString = q;
    let useRegex = Boolean(isRegex);

    if (wholeWord) {
      // force regex whole word; escape q if regex not enabled
      try {
        const escaped = useRegex ? q : escapeRegExp(q);
        searchString = `\b${escaped}\b`;
        useRegex = true;
      } catch (e: any) {
        // fallback
      }
    } else if (!useRegex) {
      // plain text search
      searchString = q;
    }

    // validate regex if using regex
    if (useRegex) {
      try {
        // attempt to construct RegExp to catch invalid patterns
        // respect matchCase option
        // eslint-disable-next-line no-new
        new RegExp(searchString, matchCase ? '' : 'i');
        setRegexError(null);
      } catch (e: any) {
        setRegexError(String(e?.message || 'Invalid regex'));
        setMatches([]);
        setDecorations((prev) => {
          try { return editor.deltaDecorations(prev, []); } catch { return []; }
        });
        return;
      }
    }

    const matchesFound = model.findMatches(searchString, true, useRegex, matchCase, null, true) || [];
    setMatches(matchesFound);

    // create decorations (yellow background)
    const newDecorations = matchesFound.map((m: any) => ({
      range: m.range,
      options: { inlineClassName: 'code-search-highlight' },
    }));

    try {
      const ids = editor.deltaDecorations(decorations, newDecorations);
      setDecorations(ids);
      if (ids.length > 0) {
        setCurrentIndex(0);
        editor.revealRange(matchesFound[0].range);
      }
    } catch (e) {
      // ignore
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedQuery, editor, monaco, options]);

  useEffect(() => () => { if (editor) { try { editor.deltaDecorations(decorations, []); } catch {} } }, [editor, decorations]);

  const goTo = (dir: 'next' | 'prev') => {
    if (!matches || matches.length === 0 || !editor) return;
    let idx = currentIndex + (dir === 'next' ? 1 : -1);
    if (idx >= matches.length) idx = 0;
    if (idx < 0) idx = matches.length - 1;
    setCurrentIndex(idx);
    const m = matches[idx];
    try {
      editor.revealRange(m.range, 1);
      editor.setSelection(m.range);
    } catch (e) {}
  };

  // keyboard shortcuts: Ctrl/Cmd+F to focus; F3 / Shift+F3 to navigate
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      // focus
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
        e.preventDefault();
        inputRef.current?.focus();
        return;
      }
      if (e.key === 'F3') {
        e.preventDefault();
        if (e.shiftKey) goTo('prev'); else goTo('next');
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [matches, editor, currentIndex, options]);

  return (
    <Box sx={{ display: 'flex', gap: 1, mb: 1, alignItems: 'center' }}>
      <TextField
        size="small"
        placeholder="Search in code..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        InputProps={{ startAdornment: <SearchIcon sx={{ mr: 1 }} /> }}
        sx={{ flex: 1 }}
        inputRef={inputRef}
        helperText={regexError ? `Regex error: ${regexError}` : `${matches.length ? `${currentIndex + 1} / ${matches.length}` : '0 matches'}`}
      />

      <ToggleButtonGroup
        size="small"
        value={[]}
        exclusive={false}
        sx={{ mr: 1 }}
      >
        <ToggleButton
          value="case"
          selected={options.matchCase}
          onChange={() => setOptions((o) => ({ ...o, matchCase: !o.matchCase }))}
          aria-label="Match case"
        >
          <Tooltip title="Match case"><TextFieldsIcon fontSize="small" /></Tooltip>
        </ToggleButton>

        <ToggleButton
          value="regex"
          selected={options.isRegex}
          onChange={() => setOptions((o) => ({ ...o, isRegex: !o.isRegex }))}
          aria-label="Regex"
        >
          <Tooltip title="Regex"><RegexIcon fontSize="small" /></Tooltip>
        </ToggleButton>

        <ToggleButton
          value="word"
          selected={options.wholeWord}
          onChange={() => setOptions((o) => ({ ...o, wholeWord: !o.wholeWord }))}
          aria-label="Whole word"
        >
          <Tooltip title="Whole word"><TitleIcon fontSize="small" /></Tooltip>
        </ToggleButton>
      </ToggleButtonGroup>

      <IconButton size="small" onClick={() => goTo('prev')} aria-label="Previous match"><ArrowUpwardIcon fontSize="small" /></IconButton>
      <IconButton size="small" onClick={() => goTo('next')} aria-label="Next match"><ArrowDownwardIcon fontSize="small" /></IconButton>
      <Typography variant="caption" sx={{ ml: 1 }}>{matches.length ? `${currentIndex + 1} / ${matches.length}` : '0'}</Typography>
    </Box>
  );
};

export default CodeSearch;
