import React, { useState } from 'react';
import { View, StyleSheet, ScrollView, Alert } from 'react-native';
import { List, Switch, Divider, Button, Avatar, Text, Subheading } from 'react-native-paper';

const SettingsScreen = ({ onLogout }) => {
  const [notifsEnabled, setNotifsEnabled] = useState(true);
  const [pushEnabled, setPushEnabled] = useState(true);
  const [syncFrequency, setSyncFrequency] = useState('Hourly');
  const [offlineMode, setOfflineMode] = useState(true);

  const handleLogout = () => {
    Alert.alert(
      "Logout",
      "Are you sure you want to log out?",
      [
        { text: "Cancel", style: "cancel" },
        { text: "Logout", onPress: onLogout, style: "destructive" }
      ]
    );
  };

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Avatar.Text size={80} label="JD" style={styles.avatar} />
        <Text style={styles.name}>John Doe</Text>
        <Text style={styles.email}>john.doe@enterprise.com</Text>
      </View>

      <List.Section>
        <List.Subheader>Notifications</List.Subheader>
        <List.Item
          title="Email Notifications"
          description="Sync status and critical alerts"
          right={() => <Switch value={notifsEnabled} onValueChange={setNotifsEnabled} color="#3f51b5" />}
        />
        <List.Item
          title="Push Notifications"
          description="Real-time mobile alerts"
          right={() => <Switch value={pushEnabled} onValueChange={setPushEnabled} color="#3f51b5" />}
        />
      </List.Section>

      <Divider />

      <List.Section>
        <List.Subheader>Sync Settings</List.Subheader>
        <List.Item
          title="Sync Frequency"
          description={syncFrequency}
          left={props => <List.Icon {...props} icon="refresh" />}
          onPress={() => {}}
        />
        <List.Item
          title="Enable Offline Mode"
          description="Access data without internet"
          right={() => <Switch value={offlineMode} onValueChange={setOfflineMode} color="#3f51b5" />}
        />
        <List.Item
          title="Primary Region"
          description="US East (N. Virginia)"
          left={props => <List.Icon {...props} icon="map-marker" />}
          onPress={() => {}}
        />
      </List.Section>

      <Divider />

      <List.Section>
        <List.Subheader>Account</List.Subheader>
        <List.Item
          title="Privacy Policy"
          left={props => <List.Icon {...props} icon="shield-check" />}
          onPress={() => {}}
        />
        <List.Item
          title="Help & Support"
          left={props => <List.Icon {...props} icon="help-circle" />}
          onPress={() => {}}
        />
      </List.Section>

      <View style={styles.footer}>
        <Button 
          mode="outlined" 
          onPress={handleLogout} 
          style={styles.logoutBtn}
          textColor="#d32f2f"
        >
          Log Out
        </Button>
        <Text style={styles.version}>Version 1.0.4 (Day 4 Release)</Text>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#FFFFFF',
  },
  header: {
    padding: 30,
    alignItems: 'center',
    backgroundColor: '#F5F7FA',
  },
  avatar: {
    backgroundColor: '#3f51b5',
    marginBottom: 15,
  },
  name: {
    fontSize: 22,
    fontWeight: 'bold',
  },
  email: {
    fontSize: 14,
    color: '#666666',
  },
  footer: {
    padding: 20,
    marginTop: 20,
    alignItems: 'center',
    marginBottom: 40,
  },
  logoutBtn: {
    borderColor: '#d32f2f',
    width: '100%',
    marginBottom: 20,
  },
  version: {
    fontSize: 12,
    color: '#9E9E9E',
  },
});

export default SettingsScreen;
