import React from 'react';
import {
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Paper,
    Chip,
    IconButton,
    Tooltip,
} from '@mui/material';
import { Check as CheckIcon, Close as CloseIcon, Visibility as ViewIcon } from '@mui/icons-material';
import { AIBusinessTermDraft } from '../../../api/catalogApi';

interface SuggestionsTableProps {
    suggestions: AIBusinessTermDraft[];
    onView: (suggestion: AIBusinessTermDraft) => void;
    onApprove: (id: string) => void;
    onReject: (id: string) => void;
}

export const SuggestionsTable: React.FC<SuggestionsTableProps> = ({
    suggestions,
    onView,
    onApprove,
    onReject,
}) => {
    return (
        <TableContainer component={Paper} elevation={0} variant="outlined">
            <Table>
                <TableHead>
                    <TableRow>
                        <TableCell>Business Term</TableCell>
                        <TableCell>Status</TableCell>
                        <TableCell>Hierarchy</TableCell>
                        <TableCell>Compliance</TableCell>
                        <TableCell align="right">Actions</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {suggestions.map((draft) => (
                        <TableRow key={draft.id} hover>
                            <TableCell component="th" scope="row">
                                <div style={{ fontWeight: 600 }}>{draft.name}</div>
                                <div style={{ fontSize: '0.875rem', color: 'gray' }}>
                                    {draft.definition.length > 50
                                        ? `${draft.definition.substring(0, 50)}...`
                                        : draft.definition}
                                </div>
                            </TableCell>
                            <TableCell>
                                <Chip
                                    label={draft.status.replace('_', ' ')}
                                    size="small"
                                    color={
                                        draft.status === 'APPROVED'
                                            ? 'success'
                                            : draft.status === 'REJECTED'
                                            ? 'error'
                                            : 'warning'
                                    }
                                />
                            </TableCell>
                            <TableCell>
                                <div style={{ fontSize: '0.875rem' }}>
                                    <div>{draft.hierarchy.level1}</div>
                                    <div style={{ color: 'gray', fontSize: '0.75rem' }}>
                                        {draft.hierarchy.level2} / {draft.hierarchy.level3}
                                    </div>
                                </div>
                            </TableCell>
                            <TableCell>
                                <div style={{ display: 'flex', gap: 4 }}>
                                    {draft.piiFlag && (
                                        <Chip label="PII" size="small" color="error" variant="outlined" />
                                    )}
                                    <Chip 
                                        label={draft.sensitivity} 
                                        size="small" 
                                        variant="outlined"
                                        color={draft.sensitivity === 'HIGH' ? 'warning' : 'default'} 
                                    />
                                    <Chip label={draft.residency} size="small" variant="outlined" />
                                </div>
                            </TableCell>
                            <TableCell align="right">
                                <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
                                    <Tooltip title="View Details">
                                        <IconButton size="small" onClick={() => onView(draft)}>
                                            <ViewIcon fontSize="small" />
                                        </IconButton>
                                    </Tooltip>
                                    {draft.status === 'DRAFT_AI' && (
                                        <>
                                            <Tooltip title="Approve">
                                                <IconButton
                                                    size="small"
                                                    color="success"
                                                    onClick={() => onApprove(draft.id)}
                                                >
                                                    <CheckIcon fontSize="small" />
                                                </IconButton>
                                            </Tooltip>
                                            <Tooltip title="Reject">
                                                <IconButton
                                                    size="small"
                                                    color="error"
                                                    onClick={() => onReject(draft.id)}
                                                >
                                                    <CloseIcon fontSize="small" />
                                                </IconButton>
                                            </Tooltip>
                                        </>
                                    )}
                                </div>
                            </TableCell>
                        </TableRow>
                    ))}
                    {suggestions.length === 0 && (
                        <TableRow>
                            <TableCell colSpan={5} align="center" style={{ padding: 40, color: 'gray' }}>
                                No suggestions found. Click "Generate" to start.
                            </TableCell>
                        </TableRow>
                    )}
                </TableBody>
            </Table>
        </TableContainer>
    );
};
