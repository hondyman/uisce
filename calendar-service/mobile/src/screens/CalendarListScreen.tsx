import React, { useState, useEffect } from 'react';
import { View, StyleSheet, FlatList, RefreshControl } from 'react-native';
import { List, FAB, Text, ActivityIndicator, Divider, Card } from 'react-native-paper';

const CalendarListScreen = () => {
  const [calendars, setCalendars] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchCalendars = async () => {
    // Simulated API call for Day 2
    setTimeout(() => {
      setCalendars([
        { id: '1', name: 'Primary Work', type: 'Google', status: 'Synced' },
        { id: '2', name: 'Microsoft Outlook', type: 'Microsoft', status: 'Disconnected' },
        { id: '3', name: 'Internal Events', type: 'Local', status: 'Synced' },
        { id: '4', name: 'Sales Team', type: 'Google', status: 'In Progress' },
      ]);
      setLoading(false);
      setRefreshing(false);
    }, 1000);
  };

  useEffect(() => {
    fetchCalendars();
  }, []);

  const onRefresh = () => {
    setRefreshing(true);
    fetchCalendars();
  };

  const renderItem = ({ item }) => (
    <Card style={styles.card} onPress={() => {}}>
      <List.Item
        title={item.name}
        description={`${item.type} • Last synced: Just now`}
        left={props => <List.Icon {...props} icon={item.type === 'Google' ? 'google' : 'microsoft-outlook'} />}
        right={props => (
          <View style={styles.statusContainer}>
            <View style={[styles.statusDot, { backgroundColor: getStatusColor(item.status) }]} />
            <Text style={styles.statusText}>{item.status}</Text>
          </View>
        )}
      />
    </Card>
  );

  const getStatusColor = (status) => {
    switch (status) {
      case 'Synced': return '#4CAF50';
      case 'Disconnected': return '#F44336';
      case 'In Progress': return '#2196F3';
      default: return '#9E9E9E';
    }
  };

  if (loading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#3f51b5" />
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <FlatList
        data={calendars}
        renderItem={renderItem}
        keyExtractor={item => item.id}
        contentContainerStyle={styles.listContent}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} colors={['#3f51b5']} />
        }
      />
      <FAB
        style={styles.fab}
        icon="plus"
        onPress={() => {}}
        label="Add Connection"
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F5F7FA',
  },
  listContent: {
    padding: 16,
  },
  card: {
    marginBottom: 12,
    elevation: 2,
    backgroundColor: '#FFFFFF',
    borderRadius: 8,
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  statusContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 8,
  },
  statusText: {
    fontSize: 12,
    color: '#666666',
  },
  fab: {
    position: 'absolute',
    margin: 16,
    right: 0,
    bottom: 0,
    backgroundColor: '#3f51b5',
  },
});

export default CalendarListScreen;
