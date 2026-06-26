import React, { useState } from 'react';
import { View, StyleSheet, KeyboardAvoidingView, Platform, TouchableOpacity } from 'react-native';
import { TextInput, Button, Title, Text, Surface } from 'react-native-paper';

const LoginScreen = ({ onLoginSuccess }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleLogin = async () => {
    setLoading(true);
    setError('');
    
    // Simulated login for Day 2
    setTimeout(() => {
      setLoading(false);
      if (email && password) {
        onLoginSuccess();
      } else {
        setError('Please enter both email and password');
      }
    }, 1500);
  };

  return (
    <KeyboardAvoidingView 
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      style={styles.container}
    >
      <Surface style={styles.surface}>
        <Title style={styles.title}>Calendar Sync</Title>
        <Text style={styles.subtitle}>Enterprise Authentication</Text>

        <TextInput
          label="Email"
          value={email}
          onChangeText={setEmail}
          mode="outlined"
          autoCapitalize="none"
          keyboardType="email-address"
          style={styles.input}
        />

        <TextInput
          label="Password"
          value={password}
          onChangeText={setPassword}
          mode="outlined"
          secureTextEntry
          style={styles.input}
        />

        {error ? <Text style={styles.error}>{error}</Text> : null}

        <Button 
          mode="contained" 
          onPress={handleLogin} 
          loading={loading}
          disabled={loading}
          style={styles.button}
        >
          Login
        </Button>

        <TouchableOpacity style={styles.ssoButton}>
          <Text style={styles.ssoText}>Sign in with SSO (SAML/OIDC)</Text>
        </TouchableOpacity>
      </Surface>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    padding: 20,
    backgroundColor: '#F5F7FA',
  },
  surface: {
    padding: 30,
    borderRadius: 12,
    elevation: 4,
    backgroundColor: '#FFFFFF',
  },
  title: {
    fontSize: 28,
    textAlign: 'center',
    fontWeight: 'bold',
    color: '#3f51b5',
  },
  subtitle: {
    textAlign: 'center',
    marginBottom: 30,
    color: '#666666',
  },
  input: {
    marginBottom: 15,
  },
  button: {
    marginTop: 10,
    paddingVertical: 5,
    backgroundColor: '#3f51b5',
  },
  error: {
    color: '#d32f2f',
    textAlign: 'center',
    marginBottom: 10,
  },
  ssoButton: {
    marginTop: 20,
    alignItems: 'center',
  },
  ssoText: {
    color: '#3f51b5',
    textDecorationLine: 'underline',
  },
});

export default LoginScreen;
