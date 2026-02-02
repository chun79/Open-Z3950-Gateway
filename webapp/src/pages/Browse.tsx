import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'

interface ScanResult {
  term: string
  count: number
}

export default function Browse() {
  const [term, setTerm] = useState('')
  const [results, setResults] = useState<ScanResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  
  const [targets, setTargets] = useState<string[]>([])
  const [targetDB, setTargetDB] = useState('LCDB')
  const [scanField, setScanField] = useState('title')
  const { token } = useAuth()

  useEffect(() => {
    fetch('/api/targets', { headers: { 'Authorization': `Bearer ${token}` } })
      .then(res => res.json())
      .then(data => {
        if (data.status === 'success' && data.data) {
          setTargets(data.data)
          if (data.data.length > 0) setTargetDB(data.data[0])
        }
      })
      .catch(console.error)
  }, [token])

  const handleScan = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setResults([])

    try {
      const response = await fetch(`/api/scan?term=${encodeURIComponent(term)}&db=${encodeURIComponent(targetDB)}&field=${encodeURIComponent(scanField)}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!response.ok) {
        throw new Error(`Error: ${response.statusText}`)
      }
      const data = await response.json()
      const list = data.data || []
      setResults(list)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const navigateToSearch = (searchTerm: string) => {
    // Navigate to Search page with pre-filled query
    // Simple way: redirect with query params. 
    // But our Search component uses internal state.
    // For now, let's just open a new tab or log.
    // Ideally, pass params via router state.
    window.location.href = `/?query=${encodeURIComponent(searchTerm)}&field=${scanField}`
  }

  return (
    <>
      <article>
        <header><strong>Browse Index (Scan)</strong></header>
        <form onSubmit={handleScan}>
          <fieldset role="group">
            <select 
              value={scanField} 
              onChange={(e) => setScanField(e.target.value)}
              style={{ width: '150px' }}
              aria-label="Scan Field"
            >
              <option value="title">Title</option>
              <option value="author">Author</option>
              <option value="subject">Subject</option>
            </select>
            <input 
              type="text" 
              name="term" 
              placeholder="Enter Start Term (e.g., Shakes)" 
              value={term}
              onChange={(e) => setTerm(e.target.value)}
              required
              style={{ flexGrow: 2 }} 
            />
            <select 
              value={targetDB} 
              onChange={(e) => setTargetDB(e.target.value)}
              style={{ flexGrow: 1 }}
              aria-label="Select Target Library"
            >
              {targets.map(t => <option key={t} value={t}>{t}</option>)}
              {targets.length === 0 && <option value="LCDB">LCDB</option>}
            </select>
            <button type="submit" disabled={loading}>
              {loading ? 'Scanning...' : 'Scan'}
            </button>
          </fieldset>
        </form>
      </article>

      {error && (
        <article className="pico-background-red-200">
          <strong>❌ Error:</strong> {error}
        </article>
      )}

      {results.length > 0 && (
        <article>
          <figure>
            <table>
              <thead>
                <tr>
                  <th scope="col">Term</th>
                  <th scope="col">Count</th>
                  <th scope="col">Action</th>
                </tr>
              </thead>
              <tbody>
                {results.map((item, index) => (
                  <tr key={index}>
                    <td>{item.term}</td>
                    <td>{item.count}</td>
                    <td>
                      <a href={`/?term1=${encodeURIComponent(item.term)}&attr1=${scanField === 'title' ? 4 : scanField === 'author' ? 1003 : 21}&db=${targetDB}`}>
                        Search
                      </a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </figure>
        </article>
      )}
      
      {results.length === 0 && !loading && !error && term && (
        <p>No index entries found.</p>
      )}
    </>
  )
}
