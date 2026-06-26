import React, { useState, useRef, useEffect } from 'react';
import { Box, Typography, TextField, IconButton, Paper, Avatar, CircularProgress, Chip, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Divider, Button, Tooltip, Zoom } from '@mui/material';
import { Send as SendIcon, SmartToy as BotIcon, Person as UserIcon, AutoAwesome as AIIcon, TableView as TableIcon, Analytics as GraphIcon, ContentCopy as CopyIcon, Download as DownloadIcon, History as HistoryIcon, Psychology as BrainIcon } from '@mui/icons-material';
import { motion, AnimatePresence } from 'framer-motion';
import { NLIntelligenceApi } from '../api/nl-intelligence';
import { NLResponse, QueryPlan } from '../types/nl-intelligence';
import { useTenant } from '../contexts/TenantContext';

interface Message {
    role: 'user' | 'assistant';
    content: string;
    result?: any;
    plan?: QueryPlan;
    intent?: string;
    reasoning?: string[];
}

const GlobalNLQueryPage: React.FC = () => {
    const { tenant } = useTenant();
    const [prompt, setPrompt] = useState('');
    const [messages, setMessages] = useState<Message[]>([
        { role: 'assistant', content: "Welcome to SemLayer Intelligence. I can help you analyze lineage, troubleshoot incidents, or query your semantic data. How can I assist you today?" }
    ]);
    const [loading, setLoading] = useState(false);
    const [reasoningStep, setReasoningStep] = useState<number>(0);
    const messagesEndRef = useRef<HTMLDivElement>(null);

    const steps = [
        "Analyzing Intent...",
        "Architecting Query Plan...",
        "Executing Governed Query...",
        "Generating Narrative Summary..."
    ];

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
        setReasoningStep(0);
        const currentPrompt = prompt;
        setPrompt('');

        try {
            // 1. Interpret
            setReasoningStep(0);
            const interpretation = await NLIntelligenceApi.interpret({ 
                question: currentPrompt, 
                tenant_scope: tenant?.id ? [tenant.id] : ['default']
            });

            // 2. Execute if we have a plan
            setReasoningStep(1);
            let resultData: any = null;
            if (interpretation.query_plan) {
                setReasoningStep(2);
                resultData = await NLIntelligenceApi.execute(interpretation.query_plan);
            }

            // 3. Summarize
            setReasoningStep(3);
            const summary = await NLIntelligenceApi.summarize(currentPrompt, resultData);

            setMessages(prev => [...prev, { 
                role: 'assistant', 
                content: summary,
                result: resultData,
                plan: interpretation.query_plan,
                intent: interpretation.intent,
                reasoning: (interpretation as any).reasoning_steps
            }]);
        } catch (err) {
            setMessages(prev => [...prev, { role: 'assistant', content: "I encountered an error while processing your request. Please try rephrasing or check your connection." }]);
        } finally {
            setLoading(false);
        }
    };

    const renderResult = (result: any) => {
        if (!result || !Array.isArray(result) || result.length === 0) return null;

        const headers = Object.keys(result[0]);
        return (
            <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2 }}>
                <TableContainer component={Paper} sx={{ mt: 2, borderRadius: 3, maxHeight: 350, overflow: 'auto', border: '1px solid rgba(0,0,0,0.05)', boxShadow: '0 4px 20px rgba(0,0,0,0.05)' }}>
                    <Box sx={{ p: 1.5, display: 'flex', justifyContent: 'flex-end', gap: 1, borderBottom: '1px solid rgba(0,0,0,0.05)', bgcolor: 'rgba(0,0,0,0.02)' }}>
                        <Button size="small" startIcon={<CopyIcon sx={{ fontSize: 14 }} />} sx={{ textTransform: 'none', borderRadius: 2 }}>Copy</Button>
                        <Button size="small" startIcon={<DownloadIcon sx={{ fontSize: 14 }} />} sx={{ textTransform: 'none', borderRadius: 2 }}>Export</Button>
                    </Box>
                    <Table size="small" stickyHeader>
                        <TableHead>
                            <TableRow>
                                {headers.map(h => (
                                    <TableCell key={h} sx={{ fontWeight: 'bold', bgcolor: '#f8fafc', color: '#64748b', fontSize: '0.75rem', textTransform: 'uppercase', py: 1 }}>{h}</TableCell>
                                ))}
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {result.slice(0, 10).map((row, i) => (
                                <TableRow key={i} sx={{ '&:hover': { bgcolor: 'rgba(33, 150, 243, 0.04)' } }}>
                                    {headers.map(h => <TableCell key={h} sx={{ fontSize: '0.875rem', py: 1 }}>{JSON.stringify(row[h])}</TableCell>)}
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                    {result.length > 10 && (
                        <Box sx={{ p: 1.5, textAlign: 'center', borderTop: '1px solid rgba(0,0,0,0.05)' }}>
                            <Typography variant="caption" color="textSecondary" fontWeight="500">Showing top 10 of {result.length} records</Typography>
                        </Box>
                    )}
                </TableContainer>
            </motion.div>
        );
    };

    return (
        <Box sx={{ height: 'calc(100vh - 64px)', display: 'flex', flexDirection: 'column', p: { xs: 1, md: 3 }, background: 'radial-gradient(circle at top left, #f3f4f6 0%, #e5e7eb 100%)', position: 'relative', overflow: 'hidden' }}>
            {/* Background Decorations */}
            <Box sx={{ position: 'absolute', top: -100, left: -100, width: 400, height: 400, background: 'rgba(33, 150, 243, 0.1)', borderRadius: '50%', filter: 'blur(80px)', pointerEvents: 'none' }} />
            <Box sx={{ position: 'absolute', bottom: -100, right: -100, width: 400, height: 400, background: 'rgba(156, 39, 176, 0.1)', borderRadius: '50%', filter: 'blur(80px)', pointerEvents: 'none' }} />

            <Paper sx={{ flex: 1, display: 'flex', flexDirection: 'column', borderRadius: 4, overflow: 'hidden', maxWidth: 1100, mx: 'auto', width: '100%', border: '1px solid rgba(255,255,255,0.4)', background: 'rgba(255,255,255,0.7)', backdropFilter: 'blur(30px)', boxShadow: '0 20px 50px rgba(0,0,0,0.1)' }}>
                {/* Header */}
                <Box sx={{ p: 2, borderBottom: '1px solid rgba(0,0,0,0.05)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', bgcolor: 'rgba(255,255,255,0.5)' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                        <Avatar sx={{ bgcolor: 'white', border: '1px solid rgba(0,0,0,0.05)', boxShadow: '0 2px 8px rgba(0,0,0,0.05)' }}>
                            <AIIcon color="primary" />
                        </Avatar>
                        <Box>
                            <Typography variant="subtitle1" fontWeight="bold" sx={{ lineHeight: 1.2 }}>Intelligence Hub</Typography>
                            <Typography variant="caption" color="textSecondary">Autonomous Semantic Reasoning</Typography>
                        </Box>
                    </Box>
                    <Box sx={{ display: 'flex', gap: 1 }}>
                        <Chip label="Enterprise AI" size="small" color="primary" sx={{ fontWeight: 'bold', fontSize: '0.65rem', borderRadius: 1 }} />
                        <Chip label="Gemini 1.5 Pro" size="small" variant="outlined" sx={{ fontWeight: 'bold', fontSize: '0.65rem', borderRadius: 1 }} />
                    </Box>
                </Box>

                {/* Chat Area */}
                <Box sx={{ flex: 1, overflowY: 'auto', p: { xs: 2, md: 4 }, display: 'flex', flexDirection: 'column', gap: 4 }}>
                    <AnimatePresence>
                        {messages.map((msg, i) => (
                            <motion.div key={i} initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.4 }} style={{ alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start', maxWidth: '85%' }}>
                                <Box sx={{ display: 'flex', gap: 2, flexDirection: msg.role === 'user' ? 'row-reverse' : 'row' }}>
                                    <Avatar sx={{ width: 36, height: 36, bgcolor: msg.role === 'user' ? 'primary.main' : 'white', border: '1px solid rgba(0,0,0,0.1)', color: msg.role === 'user' ? 'white' : 'primary.main', boxShadow: 1 }}>
                                        {msg.role === 'user' ? <UserIcon fontSize="small" /> : <BotIcon fontSize="small" />}
                                    </Avatar>
                                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: msg.role === 'user' ? 'flex-end' : 'flex-start' }}>
                                        <Paper sx={{ p: 2, borderRadius: msg.role === 'user' ? '20px 4px 20px 20px' : '4px 20px 20px 20px', bgcolor: msg.role === 'user' ? 'primary.main' : 'white', color: msg.role === 'user' ? 'white' : '#1e293b', boxShadow: '0 4px 15px rgba(0,0,0,0.05)', border: msg.role === 'user' ? 'none' : '1px solid rgba(0,0,0,0.05)' }}>
                                            <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.6, fontSize: '0.95rem' }}>{msg.content}</Typography>
                                            
                                            {msg.plan && (
                                                <Box sx={{ mt: 2, pt: 2, borderTop: '1px solid rgba(0,0,0,0.05)', display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                                                    <Tooltip title={`Engine: ${msg.plan.engine}`} TransitionComponent={Zoom}>
                                                        <Chip 
                                                            size="small" 
                                                            icon={msg.plan.type === 'SQL' ? <TableIcon sx={{ fontSize: 12 }} /> : <GraphIcon sx={{ fontSize: 12 }} />} 
                                                            label={`${msg.plan.type} Engine`}
                                                            sx={{ height: 24, fontSize: '0.7rem', fontWeight: 'bold' }}
                                                        />
                                                    </Tooltip>
                                                    {msg.plan.dialect && <Chip size="small" label={msg.plan.dialect} variant="outlined" sx={{ height: 24, fontSize: '0.7rem' }} />}
                                                    {msg.intent && <Chip size="small" label={msg.intent} sx={{ height: 24, fontSize: '0.7rem', bgcolor: 'rgba(0,0,0,0.05)' }} />}
                                                </Box>
                                            )}

                                            {msg.result && renderResult(msg.result)}
                                            
                                            {msg.reasoning && msg.reasoning.length > 0 && (
                                                <Box sx={{ mt: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                                                    <BrainIcon sx={{ fontSize: 14, color: 'text.secondary' }} />
                                                    <Typography variant="caption" color="textSecondary" sx={{ fontStyle: 'italic' }}>Chain-of-Thought reasoning applied</Typography>
                                                </Box>
                                            )}
                                        </Paper>
                                        <Typography variant="caption" color="textSecondary" sx={{ mt: 0.5, px: 1 }}>{msg.role === 'assistant' ? 'Assistant • Just now' : 'You • Just now'}</Typography>
                                    </Box>
                                </Box>
                            </motion.div>
                        ))}
                    </AnimatePresence>

                    {loading && (
                        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                            <Box sx={{ display: 'flex', gap: 2 }}>
                                <Avatar sx={{ width: 36, height: 36, bgcolor: 'white', border: '1px solid rgba(0,0,0,0.1)', color: 'secondary.main' }}>
                                    <BotIcon fontSize="small" />
                                </Avatar>
                                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                                        <CircularProgress size={16} thickness={5} />
                                        <Typography variant="body2" color="primary" fontWeight="bold">{steps[reasoningStep]}</Typography>
                                    </Box>
                                    <Paper sx={{ p: 2, borderRadius: 2, bgcolor: 'rgba(33, 150, 243, 0.05)', border: '1px dashed rgba(33, 150, 243, 0.2)' }}>
                                        <Typography variant="caption" color="textSecondary">Deep Reasoning in progress. Accessing metadata graph...</Typography>
                                    </Paper>
                                </Box>
                            </Box>
                        </motion.div>
                    )}
                    <div ref={messagesEndRef} />
                </Box>

                {/* Input Area */}
                <Box sx={{ p: 3, borderTop: '1px solid rgba(0,0,0,0.05)', background: 'white', position: 'relative' }}>
                    <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-end', maxWidth: 900, mx: 'auto' }}>
                        <TextField
                            fullWidth
                            multiline
                            maxRows={6}
                            placeholder="Ask about data, lineage, or system health..."
                            value={prompt}
                            onChange={(e) => setPrompt(e.target.value)}
                            onKeyPress={(e) => {
                                if (e.key === 'Enter' && !e.shiftKey) {
                                    e.preventDefault();
                                    handleSend();
                                }
                            }}
                            disabled={loading}
                            sx={{ '& .MuiOutlinedInput-root': { borderRadius: 3, bgcolor: '#f8fafc', '&:hover': { bgcolor: '#f1f5f9' }, '&.Mui-focused': { bgcolor: 'white' } } }}
                        />
                        <IconButton 
                            color="primary" 
                            onClick={handleSend} 
                            disabled={loading || !prompt.trim()}
                            sx={{ width: 52, height: 52, bgcolor: prompt.trim() ? 'primary.main' : 'rgba(0,0,0,0.05)', color: prompt.trim() ? 'white' : 'text.disabled', '&:hover': { bgcolor: prompt.trim() ? 'primary.dark' : 'rgba(0,0,0,0.1)' }, transition: 'all 0.2s' }}
                        >
                            <SendIcon />
                        </IconButton>
                    </Box>
                    <Box sx={{ display: 'flex', justifyContent: 'center', gap: 3, mt: 2 }}>
                        <Typography variant="caption" color="textSecondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}><BrainIcon sx={{ fontSize: 12 }} /> Governance Protected</Typography>
                        <Typography variant="caption" color="textSecondary" sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}><HistoryIcon sx={{ fontSize: 12 }} /> Dialect Aware</Typography>
                    </Box>
                </Box>
            </Paper>
        </Box>
    );
};

export default GlobalNLQueryPage;
