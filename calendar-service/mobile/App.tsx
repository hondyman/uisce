import React from 'react';
import { View, Text, StyleSheet, SafeAreaView } from 'react-native';

const App = () => {
  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Calendar Sync Mobile</Text>
      </View>
      <View style={styles.content}>
        <Text style={styles.welcome}>Welcome to your enterprise calendar sync manager.</Text>
        <Text style={styles.subtext}>Day 1: Project initialized and structure set up.</Text>
      </View>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F5F7FA',
  },
  header: {
    padding: 20,
    backgroundColor: '#3f51b5',
    alignItems: 'center',
  },
  title: {
    color: '#FFFFFF',
    fontSize: 20,
    fontWeight: 'bold',
  },
  content: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
  },
  welcome: {
    fontSize: 18,
    textAlign: 'center',
    marginBottom: 10,
    color: '#333333',
  },
  subtext: {
    fontSize: 14,
    color: '#666666',
  },
});

export default App;
