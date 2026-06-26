import React, { useState } from 'react';
import {
  Card,
  CardContent,
  CardHeader,
  TextField,
  Select,
  MenuItem,
  Button,
  Switch,
  FormControlLabel,
  Stack,
  Box,
} from '@mui/material';
import { useMutation } from '@apollo/client';
import { Zap, Clock, Bell } from 'lucide-react';
import { useForm, Controller } from 'react-hook-form';

const INSERT_TRIGGER = /* GraphQL */ `
  mutation InsertBPTrigger($object: bp_triggers_insert_input!) {
    insert_bp_triggers_one(object: $object) {
      id
    }
  }
`;

const BPTriggerBuilder: React.FC<{ processId: string }> = ({ processId }) => {
  const { control, handleSubmit } = useForm({
    defaultValues: {
      name: '',
      triggerType: 'event',
      entity: 'Order',
      action: 'created',
      priority: 5,
      enabled: true,
    },
  });
  const [triggerType, setTriggerType] = useState<string>('event');
  const [insertTrigger] = useMutation(INSERT_TRIGGER as any);

  const handleSave = async (values: any) => {
    const object: any = {
      trigger_name: values.name,
      trigger_type: triggerType,
      target_process_id: processId,
      priority: values.priority || 5,
      enabled: values.enabled,
    };
    if (triggerType === 'event') {
      object.event_config = { entity: values.entity, action: values.action };
    }

    await insertTrigger({ variables: { object } });
  };

  const triggerTypeOptions = [
    { value: 'event', label: 'Event-Driven', icon: Zap },
    { value: 'time', label: 'Scheduled', icon: Clock },
    { value: 'threshold', label: 'Threshold', icon: Bell },
  ];

  return (
    <Card>
      <CardHeader title="Create BP Trigger" />
      <CardContent>
        <Box component="form" onSubmit={handleSubmit(handleSave)} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <Controller
            name="name"
            control={control}
            rules={{ required: 'Trigger name is required' }}
            render={({ field }) => (
              <TextField
                {...field}
                label="Trigger Name"
                placeholder="e.g., VIP Order Escalation"
                variant="outlined"
                fullWidth
              />
            )}
          />

          <Controller
            name="triggerType"
            control={control}
            render={({ field }) => (
              <Select
                {...field}
                label="Trigger Type"
                onChange={(e) => {
                  field.onChange(e);
                  setTriggerType(e.target.value);
                }}
              >
                {triggerTypeOptions.map(({ value, label, icon: Icon }) => (
                  <MenuItem key={value} value={value}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Icon size={18} />
                      {label}
                    </Box>
                  </MenuItem>
                ))}
              </Select>
            )}
          />

          {triggerType === 'event' && (
            <>
              <Controller
                name="entity"
                control={control}
                render={({ field }) => (
                  <Select {...field} label="Entity">
                    <MenuItem value="Order">Order</MenuItem>
                    <MenuItem value="Employee">Employee</MenuItem>
                  </Select>
                )}
              />

              <Controller
                name="action"
                control={control}
                render={({ field }) => (
                  <Select {...field} label="Action">
                    <MenuItem value="created">Created</MenuItem>
                    <MenuItem value="updated">Updated</MenuItem>
                  </Select>
                )}
              />
            </>
          )}

          <Controller
            name="priority"
            control={control}
            render={({ field }) => (
              <TextField
                {...field}
                type="number"
                label="Priority (1-10)"
                inputProps={{ min: 1, max: 10 }}
                variant="outlined"
                fullWidth
              />
            )}
          />

          <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
            <Controller
              name="enabled"
              control={control}
              render={({ field }) => (
                <FormControlLabel
                  control={<Switch {...field} checked={field.value} />}
                  label="Enabled"
                />
              )}
            />
          </Stack>

          <Button type="submit" variant="contained" color="primary">
            Create Trigger
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
};

export default BPTriggerBuilder;
