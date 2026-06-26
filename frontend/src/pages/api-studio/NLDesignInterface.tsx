import React, { useState, useRef, useEffect } from 'react';
import { Box, Typography, TextField, Button, Paper, CircularProgress, IconButton, Avatar, Tooltip, Zoom } from '@mui/material';
import { Send as SendIcon, SmartToy as BotIcon, Person as UserIcon, AutoAwesome as AIIcon, Psychology as BrainIcon } from '@mui/icons-material';
import { motion, AnimatePresence } from 'framer-motion';
import { ApiStudioApi } from '../../api/apiStudio';
import { APIEndpoint } from '../../types/apiStudio';

interface NLDesignInterfaceProps {
    tenantId: string;
    onProposalGenerated: (proposal: APIEndpoint) => void;
}

interface Message {
    role: 'user' | 'assistant';
    content: string;
    isThinking?: boolean;
}

const NLDesignInterface: React.FC<NLDesignInterfaceProps> = ({ tenantId, onProposalGenerated }) => {
    const [prompt, setPrompt] = useState('');
    const [messages, setMessages] = useState<Message[]>([
        { role: 'assistant', content: "Hello! I'm your AI API Architect. Describe the data surface you want to expose, and I'll design a governed endpoint spec for you." }
    ]);
    const [loading, setLoading] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    const handleSend = async () => {
        if (!prompt.trim()) return;

        const userMsg: Message = { role: 'user', content: prompt };
        setMessages(prev => [...prev, userMsg]);
        setLoading(true);
        const currentPrompt = prompt;
        setPrompt('');

        try {
            const proposal = await ApiStudioApi.generateEndpointWithAI(currentPrompt, tenantId);
            setMessages(prev => [...prev, { 
                role: 'assistant', 
                content: `Architected the "${proposal.name}" endpoint. I've optimized the filters and fields based on the semantic layer mapping. You can review the full spec in the editor.` 
            }]);
            onProposalGenerated(proposal);
        } catch (err) {
            setMessages(prev => [...prev, { role: 'assistant', content: "I hit a snag while designing that endpoint. Could you provide a bit more detail on the entities involved?" }]);
        } finally {
            setLoading(false);
        }
    };

    return (
        <Paper elevation={0} sx={{ height: '100%', display: 'flex', flexDirection: 'column', borderRadius: 4, overflow: 'hidden', background: 'rgba(255, 255, 255, 0.4)', backdropFilter: 'blur(30px)', border: '1px solid rgba(255, 255, 255, 0.4)', boxShadow: '0 8px 32px rgba(0,0,0,0.05)' }}>
            <Box sx={{ p: 2, background: 'rgba(255, 255, 255, 0.6)', borderBottom: '1px solid rgba(0,0,0,0.05)', display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main', boxShadow: '0 2px 8px rgba(33, 150, 243, 0.3)' }}>
                    <AIIcon sx={{ fontSize: 18, color: 'white' }} />
                </Avatar>
                <Box>
                    <Typography variant="subtitle2" fontWeight="bold">AI Design Partner</Typography>
                    <Typography variant="caption" color="textSecondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        <BrainIcon sx={{ fontSize: 10 }} /> Governed Spec Generation
                    </Typography>
                </Box>
            </Box>

            <Box sx={{ flex: 1, overflowY: 'auto', p: { xs: 2, md: 3 }, display: 'flex', flexDirection: 'column', gap: 3 }}>
                <AnimatePresence>
                    {messages.map((msg, i) => (
                        <motion.div key={i} initial={{ opacity: 0, x: msg.role === 'user' ? 20 : -20 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: 0.3 }} style={{ alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start', maxWidth: '85%' }}>
                            <Box sx={{ display: 'flex', gap: 1.5, flexDirection: msg.role === 'user' ? 'row-reverse' : 'row' }}>
                                <Avatar sx={{ width: 28, height: 28, bgcolor: msg.role === 'user' ? 'primary.main' : 'white', border: '1px solid rgba(0,0,0,0.05)', color: msg.role === 'user' ? 'white' : 'primary.main', fontSize: '0.75rem', fontWeight: 'bold' }}>
                                    {msg.role === 'user' ? <UserIcon sx={{ fontSize: 14 }} /> : <BotIcon sx={{ fontSize: 14 }} />}
                                </Avatar>
                                <Paper sx={{ p: 1.5, borderRadius: msg.role === 'user' ? '15px 4px 15px 15px' : '4px 15px 15px 15px', bgcolor: msg.role === 'user' ? 'primary.main' : 'white', color: msg.role === 'user' ? 'white' : '#334155', boxShadow: '0 2px 10px rgba(0,0,0,0.03)', border: msg.role === 'user' ? 'none' : '1px solid rgba(0,0,0,0.05)' }}>
                                    <Typography variant="body2" sx={{ lineHeight: 1.5 }}>{msg.content}</Typography>
                                </Paper>
                            </Box>
                        </motion.div>
                    ))}
                </AnimatePresence>
                {loading && (
                    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                        <Box sx={{ display: 'flex', gap: 1.5 }}>
                            <Avatar sx={{ width: 28, height: 28, bgcolor: 'secondary.main', color: 'white' }}>
                                <BotIcon sx={{ fontSize: 14 }} />
                            </Avatar>
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, bgcolor: 'rgba(0,0,0,0.03)', px: 2, py: 1, borderRadius: 2 }}>
                                <CircularProgress size={12} thickness={6} />
                                <Typography variant="caption" fontWeight="bold" color="textSecondary">Reasoning about entities...</Typography>
                            </Box>
                        </Box>
                    </motion.div>
                )}
                <div ref={messagesEndRef} />
            </Box>

            <Box sx={{ p: 2, background: 'rgba(255, 255, 255, 0.8)', borderTop: '1px solid rgba(0,0,0,0.05)' }}>
                <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                    <TextField
                        fullWidth
                        size="small"
                        placeholder="Suggest an API for trade analysis..."
                        value={prompt}
                        onChange={(e) => setPrompt(e.target.value)}
                        onKeyPress={(e) => {
                            if (e.key === 'Enter' && !e.shiftKey) {
                                e.preventDefault();
                                handleSend();
                            }
                        }}
                        disabled={loading}
                        sx={{ '& .MuiOutlinedInput-root': { borderRadius: 3, bgcolor: '#f1f5f9', '&:hover': { bgcolor: '#e2e8f0' }, '&.Mui-focused': { bgcolor: 'white' } } }}
                    />
                    <Tooltip title="Send Request" TransitionComponent={Zoom}>
                        <IconButton color="primary" onClick={handleSend} disabled={loading || !prompt.trim()} sx={{ bgcolor: prompt.trim() ? 'primary.main' : 'transparent', color: prompt.trim() ? 'white' : 'inherit', '&:hover': { bgcolor: prompt.trim() ? 'primary.dark' : 'transparent' } }}>
                            <SendIcon sx={{ fontSize: 20 }} />
                        </IconButton>
                    </Tooltip>
                </Box>
                <Typography variant="caption" color="textSecondary" sx={{ mt: 1, display: 'block', textAlign: 'center', fontSize: '0.65rem' }}>
                    SemLayer Enterprise Design AI • Dialect Optimized
                </Typography>
            </Box>
        </Paper>
    );
};

export default NLDesignInterface;
