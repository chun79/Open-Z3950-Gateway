import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'

interface Target {
  id: number
  name: string
  host: string
  port: number
  database_name: string
  encoding: string
}

export default function Settings() {
  const [targets, setTargets] = useState<Target[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { token, user } = useAuth()

  // Form State
  const [newName, setNewName] = useState('')
  const [newHost, setNewHost] = useState('')
  const [newPort, setNewPort] = useState(210)
  const [newDB, setNewDB] = useState('')
  const [newEncoding, setNewEncoding] = useState('MARC21')
  const [testResult, setTestResult] = useState<{msg: string, type: 'success' | 'error'} | null>(null)

  const fetchTargets = async () => {
    setLoading(true)
    try {
      const response = await fetch('/api/admin/targets', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!response.ok) throw new Error("Failed to fetch targets")
      const data = await response.json()
      setTargets(data.data || [])
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure?')) return
    try {
      const response = await fetch(`/api/admin/targets/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!response.ok) throw new Error("Failed to delete")
      fetchTargets()
    } catch (err: any) {
      alert(err.message)
    }
  }

  const handleTest = async (host: string, port: number) => {
    setTestResult(null)
    try {
      const response = await fetch('/api/admin/targets/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ host, port })
      })
      const data = await response.json()
      if (data.status === 'success') {
        setTestResult({ msg: data.message, type: 'success' })
      } else {
        setTestResult({ msg: data.message, type: 'error' })
      }
    } catch (err: any) {
      setTestResult({ msg: "Request failed: " + err.message, type: 'error' })
    }
  }

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const response = await fetch('/api/admin/targets', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          name: newName,
          host: newHost,
          port: Number(newPort),
          database_name: newDB,
          encoding: newEncoding
        })
      })
      if (!response.ok) throw new Error("Failed to create")
      
      // Reset form
      setNewName('')
      setNewHost('')
      setNewPort(210)
      setNewDB('')
      setTestResult(null)
      fetchTargets()
    } catch (err: any) {
      alert(err.message)
    }
  }

  useEffect(() => {
    if (user?.role === 'admin') {
      fetchTargets()
    }
  }, [user])

  if (user?.role !== 'admin') {
    return <article>Access Denied</article>
  }

  return (
    <article>
      <header><strong>Target Management</strong></header>
      
      {error && <article className="pico-background-red-200">{error}</article>}

      <figure>
        <table role="grid">
          <thead>
            <tr>
              <th>Name</th>
              <th>Host:Port</th>
              <th>DB Name</th>
              <th>Encoding</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {targets.map(t => (
              <tr key={t.id}>
                <td><strong>{t.name}</strong></td>
                <td><small>{t.host}:{t.port}</small></td>
                <td>{t.database_name}</td>
                <td><mark>{t.encoding}</mark></td>
                <td>
                  <div role="group" style={{ marginBottom: 0 }}>
                    <button className="outline secondary" onClick={() => handleTest(t.host, t.port)} style={{ padding: '2px 8px', fontSize: '0.8em' }}>Test</button>
                    <button className="outline contrast" onClick={() => handleDelete(t.id)} style={{ padding: '2px 8px', fontSize: '0.8em' }}>Del</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </figure>

      <hr />
      
      <h5>Add New Target</h5>
      {testResult && (
        <article style={{ 
          padding: '10px', 
          fontSize: '0.9em', 
          backgroundColor: testResult.type === 'success' ? '#d4edda' : '#f8d7da',
          color: testResult.type === 'success' ? '#155724' : '#721c24',
          borderRadius: '4px',
          marginBottom: '20px'
        }}>
          {testResult.msg}
        </article>
      )}
      <form onSubmit={handleAdd}>
        <div className="grid">
          <label>Friendly Name <input value={newName} onChange={e => setNewName(e.target.value)} placeholder="e.g. British Library" required /></label>
          <label>Host Address <input value={newHost} onChange={e => setNewHost(e.target.value)} placeholder="z3950.bl.uk" required /></label>
          <label>Port <input type="number" value={newPort} onChange={e => setNewPort(Number(e.target.value))} required /></label>
        </div>
        <div className="grid">
          <label>Database Name <input value={newDB} onChange={e => setNewDB(e.target.value)} placeholder="main" required /></label>
          <label>Encoding 
            <select value={newEncoding} onChange={e => setNewEncoding(e.target.value)}>
              <option value="MARC21">MARC21 (USMARC)</option>
              <option value="UNIMARC">UNIMARC</option>
              <option value="CNMARC">CNMARC</option>
            </select>
          </label>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-end' }}>
            <button type="button" className="secondary outline" onClick={() => handleTest(newHost, newPort)} disabled={!newHost}>Test Link</button>
            <button type="submit">Add Target</button>
          </div>
        </div>
      </form>
    </article>
  )
}
