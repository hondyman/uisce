import React, { useState, useEffect } from 'react';
import { View, StyleSheet, ScrollView } from 'react-native';
import { Card, Title, Paragraph, Text, Button, ProgressBar, List, Chip, Colors } from 'react-native-paper';

const SyncStatusScreen = () => {
  const [syncs, setSyncs] = useState([]);
  const [activeSync, setActiveSync] = useState(null);

  useEffect(() => {
    // Simulated data
    setActiveSync({
      id: 's1',
      progress: 0.65,
      processed: 45,
      total: 70,
      startedAt: '10:45 AM'
    });

    setSyncs([
      { id: 'h1', time: '9:30 AM', status: 'Completed', events: 120, duration: '4.2s' },
      { id: 'h2', time: '8:30 AM', status: 'Failed', error: 'Authentication Token Expired', duration: '0.5s' },
      { id: 'h3', time: '7:30 AM', status: 'Completed', events: 15, duration: '1.8s' },
    ]);
  }, []);

  return (
    <ScrollView style={styles.container}>
      {activeSync && (
        <Card style={styles.card}>
          <Card.Content>
            <Title>Active Sync</Title>
            <View style={styles.row}>
              <Text>Progress: {Math.round(activeSync.progress * 100)}%</Text>
              <Text>{activeSync.processed}/{activeSync.total} Events</Text>
            </View>
            <ProgressBar progress={activeSync.progress} color="#3f51b5" style={styles.progress} />
            <Paragraph style={styles.time}>Started at {activeSync.startedAt}</Paragraph>
          </Card.Content>
          <Card.Actions>
            <Button color="#d32f2f">Cancel</Button>
          </Card.Actions>
        </Card>
      )}

      <Title style={styles.sectionTitle}>Recent History</Title>
      {syncs.map(sync => (
        <Card key={sync.id} style={styles.historyCard}>
          <List.Item
            title={sync.status === 'Completed' ? `Synced ${sync.events} events` : 'Sync Failed'}
            description={`${sync.time} • Duration: ${sync.duration}${sync.error ? '\n' + sync.error : ''}`}
            descriptionNumberOfLines={2}
            left={props => (
              <List.Icon 
                {...props} 
                icon={sync.status === 'Completed' ? 'check-circle' : 'alert-circle'} 
                color={sync.status === 'Completed' ? '#4CAF50' : '#F44336'} 
              />
            )}
            right={props => (
              <View style={styles.chipContainer}>
                <Chip icon={sync.status === 'Completed' ? 'check' : 'close'} textStyle={styles.chipText} style={sync.status === 'Completed' ? styles.chipSuccess : styles.chipError}>
                  {sync.status}
                </Chip>
              </View>
            )}
          />
        </Card>
      ))}
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F5F7FA',
    padding: 10,
  },
  card: {
    marginBottom: 20,
    elevation: 4,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 10,
  },
  progress: {
    height: 10,
    borderRadius: 5,
  },
  time: {
    marginTop: 10,
    fontSize: 12,
    color: '#666666',
  },
  sectionTitle: {
    marginLeft: 0,
    marginBottom: 10,
    marginTop: 10,
    fontSize: 18,
    fontWeight: 'bold',
  },
  historyCard: {
    marginBottom: 10,
    elevation: 1,
    backgroundColor: '#FFFFFF',
    borderRadius: 8,
  },
  chipContainer: {
    justifyContent: 'center',
    marginRight: 10,
  },
  chipSuccess: {
    backgroundColor: '#E8F5E9',
  },
  chipError: {
    backgroundColor: '#FFEBEE',
  },
  chipText: {
    fontSize: 10,
  }
});

export default SyncStatusScreen;
