import React, { useState } from 'react';
import { Box, TextField, Button, Typography, Paper, List, ListItem, IconButton, CircularProgress } from '@mui/material';
import { impactApi } from '../api/impactApi';
import { NodeType } from '../types';
import SendIcon from '@mui/icons-material/Send';

interface ImpactQAProps {
  nodeType: NodeType;
  nodeId: string;
  onHighlightNodes?: (nodeIds: string[]) => void;  directionMode?: 'upstream' | 'downstream' | 'both';}

interface Message {
  role: 'user' | 'assistant';
  content: string;
}

export const ImpactQA: React.FC<ImpactQAProps> = ({ nodeType, nodeId, onHighlightNodes }) => {
  const [query, setQuery] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);

  const handleSend = async () => {
    if (!query.trim()) return;

    const userMsg: Message = { role: 'user', content: query };
    setMessages(prev => [...prev, userMsg]);
    setQuery('');
    setLoading(true);

    try {
      const response = await impactApi.query(userMsg.content, { nodeType, nodeId });
      
      const answer = response.answer || response.explanation || "I processed your request but no specific answer was returned.";
      const assistantMsg: Message = { role: 'assistant', content: answer };
      setMessages(prev => [...prev, assistantMsg]);

      // If response contains highlighted node IDs, trigger the callback
      if (onHighlightNodes && response.highlightedNodeIds) {
        onHighlightNodes(response.highlightedNodeIds);
      } else if (onHighlightNodes && response.affectedNodes) {
        // Handle common variations in backend response naming
        onHighlightNodes(response.affectedNodes.map((n: any) => n.id || n));
      }
    } catch (error) {
      console.error("Failed to query impact:", error);
      setMessages(prev => [...prev, { role: 'assistant', content: "Sorry, I encountered an error answering your question." }]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box sx={{ flex: 1, overflowY: 'auto', mb: 2 }}>
        {messages.length === 0 && (
            <Typography variant="body2" color="textSecondary" align="center" sx={{ mt: 4, px: 2 }}>
                Ask me about downstream dependencies, data lineage, or potential breaking changes.
            </Typography>
        )}
        <List sx={{ p: 0 }}>
          {messages.map((msg, index) => (
            <ListItem 
              key={index} 
              sx={{ 
                flexDirection: 'column', 
                alignItems: msg.role === 'user' ? 'flex-end' : 'flex-start',
                px: 1,
                py: 0.5
              }}
            >
              <Paper 
                elevation={0}
                sx={{ 
                    p: 1.5, 
                    bgcolor: msg.role === 'user' ? '#6366f1' : '#f3f4f6', 
                    color: msg.role === 'user' ? '#fff' : '#1f2937',
                    maxWidth: '90%',
                    borderRadius: '12px',
                    borderBottomRightRadius: msg.role === 'user' ? '2px' : '12px',
                    borderBottomLeftRadius: msg.role === 'assistant' ? '2px' : '12px',
                }}
              >
                <Typography variant="body2" sx={{ lineHeight: 1.5 }}>
                  {msg.content}
                </Typography>
              </Paper>
            </ListItem>
          ))}
          {loading && (
             <ListItem sx={{ px: 1 }}>
                 <CircularProgress size={16} sx={{ color: '#6366f1' }} />
             </ListItem>
          )}
        </List>
      </Box>
      <Box sx={{ borderTop: '1px solid rgba(0,0,0,0.05)', pt: 2 }}>
        <TextField
          fullWidth
          size="small"
          variant="outlined"
          placeholder="Ask a question..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSend()}
          disabled={loading}
          autoComplete="off"
          InputProps={{
            endAdornment: (
              <IconButton 
                size="small" 
                onClick={handleSend}
                disabled={loading || !query.trim()}
                sx={{ color: '#6366f1' }}
              >
                <SendIcon fontSize="small" />
              </IconButton>
            )
          }}
          sx={{
            '& .MuiOutlinedInput-root': {
              borderRadius: '24px',
              bgcolor: '#f9fafb'
            }
          }}
        />
      </Box>
    </Box>
  );
};
