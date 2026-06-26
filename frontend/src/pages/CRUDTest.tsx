import React, { useState } from 'react';
import { devDebug } from '../utils/devLogger';
import { IconButton, Tooltip, Dialog, DialogTitle, DialogContent, DialogActions, TextField, Button } from '@mui/material';
import { Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon } from '@mui/icons-material';
import styles from './CRUDTest.module.css';

/**
 * Simple test page to verify CRUD UI components are rendering
 */
const CRUDTest: React.FC = () => {
  const [openForm, setOpenForm] = useState(false);
  const [openDelete, setOpenDelete] = useState(false);
  const [termName, setTermName] = useState('');

  return (
    <div className={styles.pageContainer}>
      <h1>CRUD Components Test</h1>
      
      <div className={styles.section}>
        <h2>Add Button Test</h2>
        <Tooltip title="Add Business Term">
          <IconButton
            size="small"
            onClick={() => {
              devDebug('[TEST] Add button clicked');
              setOpenForm(true);
            }}
            className={styles.addButton}
          >
            <AddIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <span>← Click this add button</span>
      </div>

      <div className={styles.section}>
        <h2>Edit & Delete Buttons Test</h2>
        <div className={styles.buttonsTestContainer}>
          <span>Term Name</span>
          <Tooltip title="Edit Term">
            <IconButton
              size="small"
              onClick={() => {
                devDebug('[TEST] Edit button clicked');
                setOpenForm(true);
              }}
            >
              <EditIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="Delete Term">
            <IconButton
              size="small"
              onClick={() => {
                devDebug('[TEST] Delete button clicked');
                setOpenDelete(true);
              }}
            >
              <DeleteIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </div>
      </div>

      <div className={styles.section}>
        <h2>Console Output</h2>
        <p>Open your browser console (F12) to see click logs</p>
      </div>

      {/* Test Form Dialog */}
      <Dialog open={openForm} onClose={() => setOpenForm(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create/Edit Term</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            fullWidth
            label="Term Name"
            value={termName}
            onChange={(e) => setTermName(e.target.value)}
            margin="normal"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenForm(false)}>Cancel</Button>
          <Button onClick={() => {
            devDebug('[TEST] Form saved:', termName);
            setOpenForm(false);
          }} variant="contained">
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Test Delete Dialog */}
      <Dialog open={openDelete} onClose={() => setOpenDelete(false)}>
        <DialogTitle>Delete Confirmation</DialogTitle>
        <DialogContent>
          <p>Are you sure you want to delete this term?</p>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDelete(false)}>Cancel</Button>
          <Button onClick={() => {
            devDebug('[TEST] Term deleted');
            setOpenDelete(false);
          }} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      <div className={styles.infoBox}>
        <h3>✅ If you can see:</h3>
        <ul>
          <li>Add button (➕)</li>
          <li>Edit button (✏️)</li>
          <li>Delete button (🗑️)</li>
          <li>Tooltips on hover</li>
          <li>Dialogs when clicking buttons</li>
        </ul>
        <p>Then CRUD components ARE working! The issue is with the Business Glossary Page integration.</p>
      </div>
    </div>
  );
};

export default CRUDTest;
