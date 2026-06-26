import React, { useState, useEffect } from 'react';
import { View, StyleSheet, FlatList } from 'react-native';
import { Card, Title, Text, Button, Chip, Divider, IconButton } from 'react-native-paper';

const ConflictsScreen = () => {
  const [conflicts, setConflicts] = useState([]);

  useEffect(() => {
    // Simulated data
    setConflicts([
      { 
        id: 'c1', 
        title: 'Project Update Meeting', 
        detectedAt: '10:05 AM',
        internal: '2:00 PM - 3:00 PM (Modified)',
        external: '2:30 PM - 3:30 PM (Orig)'
      },
      { 
        id: 'c2', 
        title: 'Lunch with Client', 
        detectedAt: 'Yesterday',
        internal: '12:00 PM (Deleted)',
        external: '12:00 PM (Exists)'
      }
    ]);
  }, []);

  const renderConflict = ({ item }) => (
    <Card style={styles.card}>
      <Card.Title 
        title={item.title} 
        subtitle={`Detected: ${item.detectedAt}`}
        right={(props) => <IconButton {...props} icon="information-outline" />}
      />
      <Card.Content>
        <View style={styles.row}>
          <View style={styles.side}>
            <Text style={styles.label}>Internal</Text>
            <Text style={styles.value}>{item.internal}</Text>
          </View>
          <View style={styles.divider} />
          <View style={styles.side}>
            <Text style={styles.label}>External</Text>
            <Text style={styles.value}>{item.external}</Text>
          </View>
        </View>
      </Card.Content>
      <Card.Actions style={styles.actions}>
        <Button mode="outlined" style={styles.actionBtn}>Keep Int.</Button>
        <Button mode="outlined" style={styles.actionBtn}>Keep Ext.</Button>
        <Button mode="contained" style={styles.actionBtn}>Resolve</Button>
      </Card.Actions>
    </Card>
  );

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Chip icon="alert-circle" style={styles.badge}>{conflicts.length} Pending Conflicts</Chip>
      </View>
      <FlatList
        data={conflicts}
        renderItem={renderConflict}
        keyExtractor={item => item.id}
        contentContainerStyle={styles.list}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F5F7FA',
  },
  header: {
    padding: 15,
    alignItems: 'center',
  },
  badge: {
    backgroundColor: '#FFECB3',
  },
  list: {
    padding: 10,
  },
  card: {
    marginBottom: 15,
    elevation: 3,
  },
  row: {
    flexDirection: 'row',
    marginVertical: 10,
  },
  side: {
    flex: 1,
  },
  divider: {
    width: 1,
    backgroundColor: '#E0E0E0',
    marginHorizontal: 10,
  },
  label: {
    fontSize: 12,
    color: '#666666',
    marginBottom: 4,
  },
  value: {
    fontSize: 14,
    fontWeight: '500',
  },
  actions: {
    justifyContent: 'space-between',
    paddingHorizontal: 8,
  },
  actionBtn: {
    flex: 1,
    marginHorizontal: 4,
  }
});

export default ConflictsScreen;
