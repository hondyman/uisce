import React from 'react';
import { 
  Table, TableBody, TableCell, TableHead, TableRow, 
  Paper, Chip, Button 
} from '@mui/material';

const SuggestionsTable = ({ drafts, onSelect }) => {
    return (
        <Paper>
            <Table>
                <TableHead>
                    <TableRow>
                        <TableCell>Name</TableCell>
                        <TableCell>PII</TableCell>
                        <TableCell>Sensitivity</TableCell>
                        <TableCell>Hierarchy (L1/L2/L3)</TableCell>
                        <TableCell>Status</TableCell>
                        <TableCell align="right">Actions</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {drafts.map((draft) => (
                        <TableRow key={draft.businessTermId}>
                            <TableCell>{draft.name}</TableCell>
                            <TableCell>{draft.piiFlag ? "Yes" : "No"}</TableCell>
                            <TableCell>
                                <Chip 
                                    label={draft.sensitivity} 
                                    size="small" 
                                    color={draft.sensitivity === 'HIGH' ? "error" : "default"} 
                                />
                            </TableCell>
                            <TableCell>
                                {`${draft.hierarchy.level1} / ${draft.hierarchy.level2} / ${draft.hierarchy.level3}`}
                            </TableCell>
                            <TableCell>{draft.status}</TableCell>
                            <TableCell align="right">
                                <Button 
                                    size="small" 
                                    variant="outlined" 
                                    onClick={() => onSelect(draft)}
                                >
                                    Review
                                </Button>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </Paper>
    );
};

export default SuggestionsTable;
