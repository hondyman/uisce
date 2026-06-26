import React, { useState, useEffect, useRef } from 'react';
import {
  Box,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemAvatar,
  Avatar,
  TextField,
  IconButton,
  Typography,
  Badge,
  Divider,
  InputAdornment,
  Chip,
  Menu,
  MenuItem,
} from '@mui/material';
import {
  Send as SendIcon,
  AttachFile as AttachIcon,
  MoreVert as MoreIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
import { format, formatDistanceToNow } from 'date-fns';

interface Message {
  message_id: string;
  sender_type: 'CLIENT' | 'ADVISOR';
  sender_id: string;
  message_content: string;
  is_read: boolean;
  created_at: string;
  attachments?: any[];
}

interface Thread {
  thread_id: string;
  subject: string;
  advisor_id: string;
  advisor_name?: string;
  last_message_at: string;
  unread_count_client: number;
  message_count: number;
}

export const SecureMessagingPortal: React.FC = () => {
  const [threads, setThreads] = useState<Thread[]>([]);
  const [selectedThread, setSelectedThread] = useState<Thread | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [sending, setSending] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const ws = useRef<WebSocket | null>(null);

  useEffect(() => {
    loadThreads();
    connectWebSocket();

    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, []);

  useEffect(() => {
    if (selectedThread) {
      loadMessages(selectedThread.thread_id);
    }
  }, [selectedThread]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const connectWebSocket = () => {
    const wsUrl = `wss://${window.location.host}/ws/messages`;
    ws.current = new WebSocket(wsUrl);

    ws.current.onmessage = (event) => {
      const data = JSON.parse(event.data);
      
      if (data.type === 'NEW_MESSAGE') {
        // Add message to current thread if it matches
        if (selectedThread && data.thread_id === selectedThread.thread_id) {
          setMessages((prev) => [...prev, data.message]);
        }
        
        // Update thread list
        loadThreads();
      }
    };

    ws.current.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  };

  const loadThreads = async () => {
    try {
      const response = await fetch('/api/messages/threads');
      const data = await response.json();
      setThreads(data.threads || []);
    } catch (error) {
      console.error('Failed to load threads:', error);
    }
  };

  const loadMessages = async (threadId: string) => {
    try {
      const response = await fetch(`/api/messages/threads/${threadId}/messages`);
      const data = await response.json();
      setMessages(data.messages || []);

      // Mark messages as read
      await fetch(`/api/messages/threads/${threadId}/mark-read`, {
        method: 'POST',
      });
    } catch (error) {
      console.error('Failed to load messages:', error);
    }
  };

  const sendMessage = async () => {
    if (!newMessage.trim() || !selectedThread) return;

    setSending(true);
    try {
      const response = await fetch(`/api/messages/threads/${selectedThread.thread_id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message_content: newMessage,
        }),
      });

      const data = await response.json();
      setMessages([...messages, data]);
      setNewMessage('');
      loadThreads(); // Refresh to update last message time
    } catch (error) {
      console.error('Failed to send message:', error);
    } finally {
      setSending(false);
    }
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const filteredThreads = threads.filter((thread) =>
    thread.subject.toLowerCase().includes(searchQuery.toLowerCase()) ||
    thread.advisor_name?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <Box sx={{ display: 'flex', height: '80vh', gap: 2 }}>
      {/* Thread List */}
      <Paper sx={{ width: 350, display: 'flex', flexDirection: 'column' }}>
        <Box sx={{ p: 2 }}>
          <Typography variant="h6" gutterBottom>
            Messages
          </Typography>
          <TextField
            fullWidth
            size="small"
            placeholder="Search conversations..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
          />
        </Box>

        <Divider />

        <List sx={{ flex: 1, overflow: 'auto' }}>
          {filteredThreads.map((thread) => (
            <ListItem
              key={thread.thread_id}
              button
              selected={selectedThread?.thread_id === thread.thread_id}
              onClick={() => setSelectedThread(thread)}
              sx={{
                backgroundColor: selectedThread?.thread_id === thread.thread_id ? 'action.selected' : 'inherit',
              }}
            >
              <ListItemAvatar>
                <Badge badgeContent={thread.unread_count_client} color="primary">
                  <Avatar>{thread.advisor_name?.[0] || 'A'}</Avatar>
                </Badge>
              </ListItemAvatar>
              <ListItemText
                primary={
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Typography variant="subtitle2" noWrap>
                      {thread.subject}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {formatDistanceToNow(new Date(thread.last_message_at), { addSuffix: true })}
                    </Typography>
                  </Box>
                }
                secondary={
                  <Typography variant="body2" color="text.secondary" noWrap>
                    {thread.advisor_name || 'Your Advisor'} • {thread.message_count} messages
                  </Typography>
                }
              />
            </ListItem>
          ))}
        </List>
      </Paper>

      {/* Message View */}
      <Paper sx={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {selectedThread ? (
          <>
            {/* Header */}
            <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box>
                <Typography variant="h6">{selectedThread.subject}</Typography>
                <Typography variant="body2" color="text.secondary">
                  {selectedThread.advisor_name || 'Your Advisor'}
                </Typography>
              </Box>
              <IconButton>
                <MoreIcon />
              </IconButton>
            </Box>

            {/* Messages */}
            <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
              {messages.map((message, index) => {
                const isClient = message.sender_type === 'CLIENT';
                const showDate = index === 0 || 
                  new Date(message.created_at).toDateString() !== 
                  new Date(messages[index - 1].created_at).toDateString();

                return (
                  <Box key={message.message_id}>
                    {showDate && (
                      <Box sx={{ textAlign: 'center', my: 2 }}>
                        <Chip 
                          label={format(new Date(message.created_at), 'MMMM d, yyyy')} 
                          size="small" 
                        />
                      </Box>
                    )}

                    <Box
                      sx={{
                        display: 'flex',
                        justifyContent: isClient ? 'flex-end' : 'flex-start',
                        mb: 1,
                      }}
                    >
                      <Paper
                        sx={{
                          p: 1.5,
                          maxWidth: '70%',
                          bgcolor: isClient ? 'primary.main' : 'grey.100',
                          color: isClient ? 'primary.contrastText' : 'text.primary',
                        }}
                      >
                        <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
                          {message.message_content}
                        </Typography>
                        <Typography
                          variant="caption"
                          sx={{
                            display: 'block',
                            mt: 0.5,
                            opacity: 0.7,
                          }}
                        >
                          {format(new Date(message.created_at), 'h:mm a')}
                        </Typography>
                      </Paper>
                    </Box>
                  </Box>
                );
              })}
              <div ref={messagesEndRef} />
            </Box>

            {/* Input */}
            <Box sx={{ p: 2, borderTop: 1, borderColor: 'divider' }}>
              <TextField
                fullWidth
                multiline
                maxRows={4}
                placeholder="Type your message..."
                value={newMessage}
                onChange={(e) => setNewMessage(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    sendMessage();
                  }
                }}
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton disabled>
                        <AttachIcon />
                      </IconButton>
                      <IconButton
                        color="primary"
                        onClick={sendMessage}
                        disabled={!newMessage.trim() || sending}
                      >
                        <SendIcon />
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
            </Box>
          </>
        ) : (
          <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Typography variant="body1" color="text.secondary">
              Select a conversation to start messaging
            </Typography>
          </Box>
        )}
      </Paper>
    </Box>
  );
};
