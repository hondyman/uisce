import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  CircularProgress,
  Container,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  List,
  ListItem,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Typography,
  Chip
} from '@mui/material';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import RemoveCircleOutlineIcon from '@mui/icons-material/RemoveCircleOutline';
import { useParams } from 'react-router-dom';
import { DataBundle } from '../../types/bundles';
import { RoleDetail, RoleScope, RoleStatus, RoleType } from '../../types/roles';
import { useValidationErrors } from '../../hooks/useValidationErrors';

interface RoleEditorPageProps {
  roleName?: string;
  onSave: () => void;
  onCancel: () => void;
}

const roleStatusOptions: RoleStatus[] = ['Draft', 'Active', 'Suspended', 'Retired'];
const roleTypeOptions: RoleType[] = ['Business', 'System', 'Technical'];
const roleScopeOptions: RoleScope[] = ['Global', 'Tenant', 'Environment'];

const RoleEditorPage: React.FC<RoleEditorPageProps> = ({ roleName: propRoleName, onSave, onCancel }) => {
  const params = useParams<{ roleName?: string }>();
  const resolvedRoleName = propRoleName ?? params.roleName ?? '';
  const isEditMode = resolvedRoleName.length > 0;

  const [name, setName] = useState<string>(resolvedRoleName);
  const [description, setDescription] = useState<string>('');
  const [initialDescription, setInitialDescription] = useState<string>('');

  const [status, setStatus] = useState<RoleStatus>('Draft');
  const [initialStatus, setInitialStatus] = useState<RoleStatus>('Draft');

  const [roleType, setRoleType] = useState<RoleType>('Business');
  const [initialRoleType, setInitialRoleType] = useState<RoleType>('Business');

  const [scope, setScope] = useState<RoleScope>('Global');
  const [initialScope, setInitialScope] = useState<RoleScope>('Global');

  const [owner, setOwner] = useState<string>('');
  const [initialOwner, setInitialOwner] = useState<string>('');

  const [tagsInput, setTagsInput] = useState<string>('');
  const [initialTags, setInitialTags] = useState<string[]>([]);

  const [statusNotes, setStatusNotes] = useState<string>('');

  const [allBundles, setAllBundles] = useState<DataBundle[]>([]);
  const [assignedBundleIDs, setAssignedBundleIDs] = useState<Set<string>>(new Set<string>());
  const [initialAssignedBundleIDs, setInitialAssignedBundleIDs] = useState<Set<string>>(new Set<string>());

  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const {
    clearFieldErrors,
    clearFieldError,
    fieldHelperText,
    hasFieldError,
    handleResponseError
  } = useValidationErrors();

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);
        clearFieldErrors();

        const bundlesRes = await fetch('/api/bundles', { credentials: 'include' });
        if (!bundlesRes.ok) {
          throw new Error('Failed to fetch bundles');
        }
        const bundlesData: DataBundle[] = await bundlesRes.json();
        setAllBundles(bundlesData);

        if (isEditMode) {
          const targetRoleName = resolvedRoleName;
          const roleRes = await fetch(`/api/roles/${encodeURIComponent(targetRoleName)}`, { credentials: 'include' });
          if (!roleRes.ok) {
            const message = await roleRes.text();
            throw new Error(message || `Failed to fetch role ${targetRoleName}`);
          }
          const roleData: RoleDetail = await roleRes.json();

          setName(roleData?.name ?? targetRoleName);
          setDescription(roleData?.description ?? '');
          setInitialDescription(roleData?.description ?? '');

          setStatus(roleData?.status ?? 'Draft');
          setInitialStatus(roleData?.status ?? 'Draft');

          setRoleType(roleData?.type ?? 'Business');
          setInitialRoleType(roleData?.type ?? 'Business');

          setScope(roleData?.scope ?? 'Global');
          setInitialScope(roleData?.scope ?? 'Global');

          setOwner(roleData?.owner ?? '');
          setInitialOwner(roleData?.owner ?? '');

          const tags = roleData?.tags ?? [];
          setTagsInput(tags.join(', '));
          setInitialTags(tags);

          const assignedRes = await fetch(`/api/roles/${encodeURIComponent(targetRoleName)}/bundles`, { credentials: 'include' });
          if (!assignedRes.ok) {
            const message = await assignedRes.text();
            throw new Error(message || 'Failed to fetch assigned bundles');
          }
          const assignedData: string[] = await assignedRes.json();
          const assignedSet = new Set<string>(assignedData);
          setAssignedBundleIDs(assignedSet);
          setInitialAssignedBundleIDs(new Set<string>(assignedSet));
        } else {
          setDescription('');
          setInitialDescription('');
          setStatus('Draft');
          setInitialStatus('Draft');
          setRoleType('Business');
          setInitialRoleType('Business');
          setScope('Global');
          setInitialScope('Global');
          setOwner('');
          setInitialOwner('');
          setTagsInput('');
          setInitialTags([]);
          setAssignedBundleIDs(new Set<string>());
          setInitialAssignedBundleIDs(new Set<string>());
        }
      } catch (err: any) {
        setError(err?.message ?? 'Failed to load role editor.');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [resolvedRoleName, isEditMode, clearFieldErrors]);

  const availableBundles = useMemo(
    () => allBundles.filter((bundle) => !assignedBundleIDs.has(bundle.id)),
    [allBundles, assignedBundleIDs]
  );
  const assignedBundles = useMemo(
    () => allBundles.filter((bundle) => assignedBundleIDs.has(bundle.id)),
    [allBundles, assignedBundleIDs]
  );

  const bundleAssignmentError = fieldHelperText('bundleAssignments');

  const handleAssignBundle = (bundleId: string) => {
    clearFieldError('bundleAssignments');
    setAssignedBundleIDs((prev) => {
      const next = new Set(prev);
      next.add(bundleId);
      return next;
    });
  };

  const handleUnassignBundle = (bundleId: string) => {
    clearFieldError('bundleAssignments');
    setAssignedBundleIDs((prev) => {
      const next = new Set(prev);
      next.delete(bundleId);
      return next;
    });
  };

  const parseTags = (input: string) =>
    input
      .split(',')
      .map((tag) => tag.trim())
      .filter((tag) => tag.length > 0);

  const tagsEqual = (a: string[], b: string[]) => {
    if (a.length !== b.length) {
      return false;
    }
    const left = [...a].sort();
    const right = [...b].sort();
    return left.every((value, index) => value === right[index]);
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    clearFieldErrors();

    try {
      const trimmedName = name.trim();
      const trimmedDescription = description.trim();
      const normalizedTags = parseTags(tagsInput);
      const trimmedOwner = owner.trim();

      let effectiveRoleName = resolvedRoleName;

      if (!isEditMode) {
        const createResponse = await fetch('/api/roles', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          credentials: 'include',
          body: JSON.stringify({
            name: trimmedName,
            description: trimmedDescription,
            type: roleType,
            scope,
            tags: normalizedTags
          })
        });

        if (!createResponse.ok) {
          await handleResponseError(createResponse, 'Failed to create role.');
          setSaving(false);
          return;
        }

        const createdRole: RoleDetail = await createResponse.json();
        effectiveRoleName = createdRole.name;
        setName(createdRole.name);
        setInitialDescription(createdRole.description ?? trimmedDescription);
        setInitialStatus(createdRole.status ?? 'Draft');
        setInitialRoleType(createdRole.type ?? roleType);
        setInitialScope(createdRole.scope ?? scope);
        setInitialTags(createdRole.tags ?? normalizedTags);
        setInitialOwner(createdRole.owner ?? trimmedOwner);
      } else if (isEditMode && effectiveRoleName) {
        const updatePayload: Record<string, unknown> = {};

        if (trimmedDescription !== initialDescription) {
          updatePayload.description = trimmedDescription;
        }
        if (status !== initialStatus) {
          updatePayload.status = status;
          if (statusNotes.trim().length > 0) {
            updatePayload.notes = statusNotes.trim();
          }
        } else if (statusNotes.trim().length > 0) {
          updatePayload.notes = statusNotes.trim();
        }
        if (roleType !== initialRoleType) {
          updatePayload.type = roleType;
        }
        if (scope !== initialScope) {
          updatePayload.scope = scope;
        }
        if (!tagsEqual(normalizedTags, initialTags)) {
          updatePayload.tags = normalizedTags;
        }
        if (trimmedOwner !== initialOwner.trim() && trimmedOwner !== '') {
          updatePayload.owner = trimmedOwner;
        }

        if (Object.keys(updatePayload).length > 0) {
          const updateResponse = await fetch(`/api/roles/${encodeURIComponent(effectiveRoleName)}`, {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json'
            },
            credentials: 'include',
            body: JSON.stringify(updatePayload)
          });

          if (!updateResponse.ok) {
            if (updateResponse.status !== 404) {
              await handleResponseError(updateResponse, 'Failed to update role.');
            }
            setSaving(false);
            return;
          }

          const updatedRole: RoleDetail = await updateResponse.json();
          setInitialDescription(updatedRole.description ?? trimmedDescription);
          setInitialStatus(updatedRole.status ?? status);
          setInitialRoleType(updatedRole.type ?? roleType);
          setInitialScope(updatedRole.scope ?? scope);
          setInitialTags(updatedRole.tags ?? normalizedTags);
          setInitialOwner(updatedRole.owner ?? trimmedOwner);
          setStatus(updatedRole.status ?? status);
          setRoleType(updatedRole.type ?? roleType);
          setScope(updatedRole.scope ?? scope);
          setTagsInput((updatedRole.tags ?? normalizedTags).join(', '));
          setStatusNotes('');
        }
      }

      if (!effectiveRoleName) {
        throw new Error('Role name is required.');
      }

      const targetRole = encodeURIComponent(effectiveRoleName);
      const currentAssignments = Array.from(assignedBundleIDs);
      const initialAssignments = Array.from(initialAssignedBundleIDs);

      const toAssign = currentAssignments.filter((bundleId) => !initialAssignedBundleIDs.has(bundleId));
      const toUnassign = initialAssignments.filter((bundleId) => !assignedBundleIDs.has(bundleId));

      for (const bundleId of toAssign) {
        const assignResponse = await fetch(`/api/roles/${targetRole}/bundles`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          credentials: 'include',
          body: JSON.stringify({ bundleId })
        });

        if (!assignResponse.ok) {
          await handleResponseError(assignResponse, 'Failed to assign bundle to role.');
          setSaving(false);
          return;
        }
      }

      for (const bundleId of toUnassign) {
        const unassignResponse = await fetch(`/api/roles/${targetRole}/bundles/${encodeURIComponent(bundleId)}`, {
          method: 'DELETE',
          credentials: 'include'
        });

        if (!unassignResponse.ok) {
          await handleResponseError(unassignResponse, 'Failed to unassign bundle from role.');
          setSaving(false);
          return;
        }
      }

      setInitialAssignedBundleIDs(new Set<string>(assignedBundleIDs));
      onSave();
    } catch (err: any) {
      setError(err?.message ?? 'Failed to save role.');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Container maxWidth="xl" sx={{ mt: 4 }}>
      <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 2 }}>
        <Typography variant="h4" gutterBottom>
          {isEditMode ? `Edit Role: ${name || resolvedRoleName}` : 'Create Role'}
        </Typography>
        {isEditMode && (
          <Chip size="small" color="primary" label={status} />
        )}
      </Stack>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      <Paper sx={{ p: 3, mb: 3 }}>
        <Grid container spacing={2}>
          <Grid item xs={12} md={6}>
            <TextField
              label="Role Name"
              value={name}
              onChange={(event) => {
                clearFieldError('name');
                setName(event.target.value);
              }}
              fullWidth
              sx={{ mb: 2 }}
              disabled={isEditMode}
              error={hasFieldError('name')}
              helperText={fieldHelperText('name')}
              required={!isEditMode}
            />
          </Grid>
          <Grid item xs={12} md={6}>
            <TextField
              label="Role Owner"
              value={owner}
              onChange={(event) => {
                clearFieldError('owner');
                setOwner(event.target.value);
              }}
              fullWidth
              disabled={!isEditMode}
              helperText={isEditMode ? 'Optional: reassign stewardship for this role.' : 'Owner is set automatically on creation.'}
            />
          </Grid>
          <Grid item xs={12}>
            <TextField
              label="Description"
              value={description}
              onChange={(event) => {
                clearFieldError('description');
                setDescription(event.target.value);
              }}
              fullWidth
              multiline
              rows={2}
              error={hasFieldError('description')}
              helperText={fieldHelperText('description')}
            />
          </Grid>
        </Grid>
      </Paper>

      <Grid container spacing={3}>
        <Grid item xs={12} md={4}>
          <Card>
            <CardHeader title="Classification" subheader="Control the lifecycle and taxonomy of this role." />
            <Divider />
            <CardContent>
              <Stack spacing={2}>
                <FormControl fullWidth>
                  <InputLabel id="role-type-label">Role Type</InputLabel>
                  <Select
                    labelId="role-type-label"
                    label="Role Type"
                    value={roleType}
                    onChange={(event) => setRoleType(event.target.value as RoleType)}
                  >
                    {roleTypeOptions.map((option) => (
                      <MenuItem key={option} value={option}>
                        {option}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <FormControl fullWidth>
                  <InputLabel id="role-scope-label">Scope</InputLabel>
                  <Select
                    labelId="role-scope-label"
                    label="Scope"
                    value={scope}
                    onChange={(event) => setScope(event.target.value as RoleScope)}
                  >
                    {roleScopeOptions.map((option) => (
                      <MenuItem key={option} value={option}>
                        {option}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <FormControl fullWidth disabled={!isEditMode}>
                  <InputLabel id="role-status-label">Lifecycle Status</InputLabel>
                  <Select
                    labelId="role-status-label"
                    label="Lifecycle Status"
                    value={status}
                    onChange={(event) => setStatus(event.target.value as RoleStatus)}
                  >
                    {roleStatusOptions.map((option) => (
                      <MenuItem key={option} value={option}>
                        {option}
                      </MenuItem>
                    ))}
                  </Select>
                  {!isEditMode && (
                    <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5 }}>
                      New roles start as Draft and can be promoted after creation.
                    </Typography>
                  )}
                </FormControl>

                {isEditMode && (
                  <TextField
                    label="Status Notes"
                    value={statusNotes}
                    onChange={(event) => setStatusNotes(event.target.value)}
                    multiline
                    minRows={2}
                    placeholder="Document why the status is changing."
                  />
                )}

                <TextField
                  label="Tags"
                  value={tagsInput}
                  onChange={(event) => {
                    clearFieldError('tags');
                    setTagsInput(event.target.value);
                  }}
                  helperText="Comma-separated tags used for governance filters."
                  error={hasFieldError('tags')}
                />
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={8}>
          <Card>
            <CardHeader title="Bundle Assignments" subheader="Bind data bundles that this role should manage." />
            <Divider />
            <CardContent>
              {bundleAssignmentError && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  {bundleAssignmentError}
                </Alert>
              )}
              <Grid container spacing={2}>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle1" gutterBottom>
                    Available Bundles
                  </Typography>
                  <List dense sx={{ maxHeight: 360, overflow: 'auto', border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
                    {availableBundles.map((bundle) => (
                      <ListItem
                        key={bundle.id}
                        secondaryAction={
                          <IconButton edge="end" onClick={() => handleAssignBundle(bundle.id)}>
                            <AddCircleOutlineIcon color="primary" />
                          </IconButton>
                        }
                      >
                        <ListItemText primary={bundle.name} secondary={`v${bundle.version}`} />
                      </ListItem>
                    ))}
                    {availableBundles.length === 0 && (
                      <ListItem>
                        <ListItemText
                          primary="All bundles are currently assigned."
                          secondary="Unassign a bundle to make it available again."
                        />
                      </ListItem>
                    )}
                  </List>
                </Grid>

                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle1" gutterBottom>
                    Assigned Bundles
                  </Typography>
                  <List dense sx={{ maxHeight: 360, overflow: 'auto', border: '1px solid', borderColor: 'divider', borderRadius: 1 }}>
                    {assignedBundles.map((bundle) => (
                      <ListItem
                        key={bundle.id}
                        secondaryAction={
                          <IconButton edge="end" onClick={() => handleUnassignBundle(bundle.id)}>
                            <RemoveCircleOutlineIcon color="error" />
                          </IconButton>
                        }
                      >
                        <ListItemText primary={bundle.name} secondary={`v${bundle.version}`} />
                      </ListItem>
                    ))}
                    {assignedBundles.length === 0 && (
                      <ListItem>
                        <ListItemText
                          primary="No bundles assigned yet."
                          secondary="Assign bundles to grant this role access."
                        />
                      </ListItem>
                    )}
                  </List>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Box sx={{ mt: 4, display: 'flex', justifyContent: 'flex-end' }}>
        <Button onClick={onCancel} sx={{ mr: 2 }} disabled={saving}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={saving}
          startIcon={saving ? <CircularProgress size={18} color="inherit" /> : undefined}
        >
          {saving ? 'Saving…' : 'Save Changes'}
        </Button>
      </Box>
    </Container>
  );
};

export default RoleEditorPage;
