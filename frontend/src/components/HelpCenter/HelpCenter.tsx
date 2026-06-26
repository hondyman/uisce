
import React, { useState } from 'react';
import { 
  Box, 
  Drawer, 
  Typography, 
  List, 
  ListItem, 
  ListItemText, 
  ListItemIcon, 
  Divider, 
  IconButton,
  Button,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Card,
  Dialog,
  DialogTitle,
  DialogContent
} from '@mui/material';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import CloseIcon from '@mui/icons-material/Close';
import ArticleIcon from '@mui/icons-material/Article';
import PlayLessonIcon from '@mui/icons-material/PlayLesson';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import SchoolIcon from '@mui/icons-material/School';
import VideoLibraryIcon from '@mui/icons-material/VideoLibrary';

import ReactMarkdown from 'react-markdown';
import { PIPELINE_NODES_DOC } from './docs/pipelineNodes';

interface HelpContext {
    id: string;
    title: string;
    articles: { title: string; summary: string; docContent?: string }[];
    tutorials: { title: string; duration: string; videoUrl: string }[];
}

const helpContexts: Record<string, HelpContext> = {
    'workflow-studio': {
        id: 'workflow-studio',
        title: 'Workflow Studio Help',
        articles: [
            { title: 'Understanding Nodes', summary: 'Learn about the different types of nodes: Action, Trigger, and Logic.', docContent: PIPELINE_NODES_DOC },
            { title: 'Connecting Components', summary: 'How to drag and drop connections between nodes to define data flow.', docContent: PIPELINE_NODES_DOC }, // Using same doc for now
            { title: 'Using Templates', summary: 'Jumpstart your design using pre-built industry standard templates.', docContent: PIPELINE_NODES_DOC },
        ],
        tutorials: [
            { title: 'Building your first pipeline', duration: '5 min', videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ' }, 
            { title: 'Debugging with Trace View', duration: '3 min', videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ' },
        ]
    },
    'rules-editor': {
        id: 'rules-editor',
        title: 'Rule Editor Help',
        articles: [
            { title: 'Writing Natural Language Rules', summary: 'Tips for phrasing your requirements for the AI generator.' },
            { title: 'Understanding Rego', summary: 'A brief introduction to the Open Policy Agent (OPA) policy language.' },
            { title: 'Testing Policies', summary: 'How to use the dry-run feature to verify your rules against sample data.' },
        ],
        tutorials: [
            { title: 'Creating an Approval Policy', duration: '4 min', videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ' },
            { title: 'Advanced Logic & Exceptions', duration: '6 min', videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ' },
        ]
    }
};

interface HelpCenterProps {
    context: 'workflow-studio' | 'rules-editor';
}

export const HelpCenter: React.FC<HelpCenterProps> = ({ context }) => {
    const [open, setOpen] = useState(false);
    const [videoOpen, setVideoOpen] = useState(false);
    const [docsOpen, setDocsOpen] = useState(false);
    const [currentVideoUrl, setCurrentVideoUrl] = useState('');
    const [currentDocContent, setCurrentDocContent] = useState('');
    
    const data = helpContexts[context];

    const handlePlayVideo = (url: string) => {
        setCurrentVideoUrl(url);
        setVideoOpen(true);
    };

    const handleOpenDocs = (content?: string) => {
        if (content) {
            setCurrentDocContent(content);
            setDocsOpen(true);
        } else {
            // Fallback for external link if no content
            window.open('https://docs.semlayer.io', '_blank');
        }
    };

    return (
        <>
            <IconButton 
                onClick={() => setOpen(true)}
                sx={{ 
                    position: 'fixed', 
                    bottom: 24, 
                    right: 24, 
                    bgcolor: 'primary.main', 
                    color: 'white',
                    boxShadow: 3,
                    width: 56,
                    height: 56,
                    '&:hover': { bgcolor: 'primary.dark' },
                    zIndex: 1300
                }}
            >
                <HelpOutlineIcon fontSize="large" />
            </IconButton>

            <Drawer
                anchor="right"
                open={open}
                onClose={() => setOpen(false)}
                PaperProps={{ sx: { width: 380, bgcolor: '#f8fafc' } }}
            >
                <Box sx={{ p: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid #e2e8f0', bgcolor: 'white' }}>
                    <Box display="flex" alignItems="center" gap={1}>
                        <SchoolIcon color="primary" />
                        <Typography variant="h6" fontWeight="bold" color="#1e293b">
                            Learning Center
                        </Typography>
                    </Box>
                    <IconButton size="small" onClick={() => setOpen(false)}>
                        <CloseIcon />
                    </IconButton>
                </Box>

                <Box sx={{ p: 3 }}>
                    <Typography variant="subtitle2" fontWeight="bold" sx={{ mb: 2, color: '#64748b', textTransform: 'uppercase', fontSize: '0.75rem' }}>
                        {data.title}
                    </Typography>

                    <Box sx={{ mb: 4 }}>
                        <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold' }}>Quick Tutorials</Typography>
                        {data.tutorials.map((tutorial, idx) => (
                            <Card 
                                key={idx} 
                                variant="outlined" 
                                onClick={() => handlePlayVideo(tutorial.videoUrl)}
                                sx={{ mb: 2, bgcolor: 'white', '&:hover': { borderColor: 'primary.main', cursor: 'pointer' } }}
                            >
                                <ListItem>
                                    <ListItemIcon sx={{ minWidth: 40 }}>
                                        <PlayLessonIcon color="secondary" />
                                    </ListItemIcon>
                                    <ListItemText 
                                        primary={tutorial.title} 
                                        secondary={
                                            <Chip 
                                                label={tutorial.duration} 
                                                size="small" 
                                                icon={<VideoLibraryIcon sx={{ fontSize: '0.9rem !important' }} />} 
                                                sx={{ mt: 0.5, height: 20, fontSize: '0.7rem' }} 
                                            />
                                        } 
                                    />
                                </ListItem>
                            </Card>
                        ))}
                    </Box>

                    <Divider sx={{ mb: 4 }} />

                    <Typography variant="h6" sx={{ mb: 2, fontWeight: 'bold' }}>Documentation</Typography>
                    {data.articles.map((article, idx) => (
                         <Accordion key={idx} disableGutters elevation={0} sx={{ '&:before': { display: 'none' }, mb: 1, border: '1px solid #e2e8f0', borderRadius: '8px !important' }}>
                             <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                                 <Typography fontWeight="medium">{article.title}</Typography>
                             </AccordionSummary>
                             <AccordionDetails sx={{ bgcolor: '#f1f5f9', borderTop: '1px solid #e2e8f0' }}>
                                 <Typography variant="body2" color="text.secondary">
                                     {article.summary}
                                 </Typography>
                                 <Button 
                                    size="small" 
                                    sx={{ mt: 1 }} 
                                    startIcon={<ArticleIcon />}
                                    onClick={() => handleOpenDocs(article.docContent)}
                                 >
                                    Read Guide
                                </Button>
                             </AccordionDetails>
                         </Accordion>
                    ))}

                    <Box sx={{ mt: 4, p: 2, bgcolor: '#eff6ff', borderRadius: 2, border: '1px dashed #bfdbfe' }}>
                        <Typography variant="subtitle2" color="primary.dark" align="center">
                            Need more help?
                        </Typography>
                        <Button fullWidth variant="outlined" size="small" sx={{ mt: 1 }}>
                            Contact Support
                        </Button>
                    </Box>

                </Box>
            </Drawer>

            <Dialog 
                open={videoOpen} 
                onClose={() => setVideoOpen(false)} 
                maxWidth="md" 
                fullWidth
                PaperProps={{ sx: { bgcolor: 'black', borderRadius: 2 } }}
            >
                <DialogTitle sx={{ color: 'white', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    Video Tutorial
                    <IconButton size="small" onClick={() => setVideoOpen(false)} sx={{ color: 'rgba(255,255,255,0.7)' }}>
                        <CloseIcon />
                    </IconButton>
                </DialogTitle>
                <DialogContent sx={{ p: 0 }}>
                     <Box sx={{ position: 'relative', paddingTop: '56.25%', bgcolor: 'black' }}>
                       <iframe 
                         style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%' }}
                         src={currentVideoUrl} 
                         frameBorder="0" 
                         allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                         allowFullScreen 
                       />
                     </Box>
                </DialogContent>
            </Dialog>

            <Dialog
                open={docsOpen}
                onClose={() => setDocsOpen(false)}
                maxWidth="md"
                fullWidth
                scroll="paper"
            >
                <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    Documentation
                    <IconButton onClick={() => setDocsOpen(false)}>
                        <CloseIcon />
                    </IconButton>
                </DialogTitle>
                <DialogContent dividers>
                    <Box sx={{ 
                        '& h1': { fontSize: '1.5rem', fontWeight: 'bold', mb: 2, color: 'primary.main' },
                        '& h2': { fontSize: '1.25rem', fontWeight: 'bold', mt: 3, mb: 1, borderBottom: '1px solid #eaeaea', pb: 0.5 },
                        '& h3': { fontSize: '1.1rem', fontWeight: 'bold', mt: 2, mb: 1 },
                        '& p': { mb: 1.5, lineHeight: 1.6 },
                        '& ul': { pl: 3, mb: 1.5 },
                        '& li': { mb: 0.5 },
                        '& code': { bgcolor: '#f1f5f9', px: 0.5, py: 0.2, borderRadius: 1, fontFamily: 'monospace', fontSize: '0.9em' }
                    }}>
                        <ReactMarkdown>{currentDocContent}</ReactMarkdown>
                    </Box>
                </DialogContent>
            </Dialog>
        </>
    );
};


export default HelpCenter;
