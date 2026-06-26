import type React from 'react'
import { useEffect, useState } from 'react'
import { devDebug, devError } from '../../../src/utils/devLogger';
import { Button } from '@mui/material'
import { useNavigate } from 'react-router-dom';
const styles: { [key: string]: string } = {
  pageContainer: '',
  configButton: '',
};

export default function UnifiedCRUDPage() {
  const [status, setStatus] = useState('idle')
  const navigate = useNavigate();

  useEffect(() => {
    // noop
  }, [])

  const smokeTest = async () => {
    setStatus('testing')
    try {
      const resp = await fetch('/api/health')
      if (resp.ok) setStatus('ok')
      else setStatus('bad')
    } catch (err) {
      setStatus('error')
    }
  }

  const _doDynamicInsert = async () => {
    setStatus('inserting')
    try {
      const payload = {
        input: {
          entity_type: 'client_investors',
          object: { type: 'ClientInvestor', name: 'From Frontend', custom_fields: {} }
        }
      }
      const res = await fetch('/api/actions/dynamic-insert', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      const data = await res.json()
      devDebug('action response', data)
      if (res.ok) setStatus('inserted')
      else setStatus('insert-failed')
    } catch (e) {
      devError(e)
      setStatus('error')
    }
  }

  // intentionally reference the helper to silence "declared but never used" diagnostics
  void _doDynamicInsert;

  return (
    <div className={styles.pageContainer}>
      <h2>Unified CRUD (local)</h2>
      <p>Backend health: {status}</p>
      <Button onClick={smokeTest}>Run backend health check</Button>
      <Button onClick={() => navigate('/config')} className={styles.configButton}>
        🔧 Config
      </Button>
    </div>
  )
}
