import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'

interface ScanResult {
  term: string
  count: number
}

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ".split("")

export default function Browse() {
  const [term, setTerm] = useState('')
  const [results, setResults] = useState<ScanResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  
  const [targets, setTargets] = useState<string[]>([])
  const [targetDB, setTargetDB] = useState('LCDB')
  const [scanField, setScanField] = useState('title')
  const { token } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()

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

  const executeScan = async (scanTerm: string) => {
    setLoading(true)
    setError('')
    setResults([])

    try {
      const response = await fetch(`/api/scan?term=${encodeURIComponent(scanTerm)}&db=${encodeURIComponent(targetDB)}&field=${encodeURIComponent(scanField)}`, {
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

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    executeScan(term)
  }

  const handleAlphaClick = (letter: string) => {
    setTerm(letter)
    executeScan(letter)
  }

  const handleTermClick = (item: ScanResult) => {
    // Map scan field to search attribute
    // Title -> 4, Author -> 1003, Subject -> 21
    let attr = '4'
    if (scanField === 'author') attr = '1003'
    if (scanField === 'subject') attr = '21'
    
    // Navigate to Search with query params
    navigate(`/?term=${encodeURIComponent(item.term)}&attr=${attr}&db=${targetDB}`)
  }

  return (
    <>
      <article>
        <header>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <strong>{t('browse.title')}</strong>
            <select 
              value={targetDB} 
              onChange={(e) => setTargetDB(e.target.value)}
              style={{ width: 'auto', marginBottom: 0 }}
            >
              {targets.map(t => <option key={t} value={t}>{t}</option>)}
              {targets.length === 0 && <option value="LCDB">LCDB</option>}
            </select>
          </div>
        </header>

        <form onSubmit={handleFormSubmit}>
          <div className="grid">
            <select 
              value={scanField} 
              onChange={(e) => setScanField(e.target.value)}
              aria-label="Scan Field"
            >
              <option value="title">{t('browse.by_title')}</option>
              <option value="author">{t('browse.by_author')}</option>
              <option value="subject">{t('browse.by_subject')}</option>
            </select>
            <input 
              type="text" 
              placeholder={t('browse.placeholder')} 
              value={term}
              onChange={(e) => setTerm(e.target.value)}
              required
            />
            <button type="submit" disabled={loading}>
              {loading ? t('browse.scanning') : t('browse.button')}
            </button>
          </div>
        </form>

        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '5px', justifyContent: 'center', marginTop: '10px' }}>
          {ALPHABET.map(char => (
            <button 
              key={char} 
              className="outline secondary" 
              style={{ padding: '5px 10px', fontSize: '0.8em', marginBottom: 0 }}
              onClick={() => handleAlphaClick(char)}
            >
              {char}
            </button>
          ))}
        </div>
      </article>

      {error && (
        <article className="pico-background-red-200">
          <strong>❌ {t('search.error')}:</strong> {error}
        </article>
      )}

      {results.length > 0 && (
        <div className="grid">
          {results.map((item, index) => (
            <article key={index} className="scan-card" onClick={() => handleTermClick(item)} style={{ cursor: 'pointer', marginBottom: '10px' }}>
              <header style={{ padding: '10px' }}>
                <strong>{item.term}</strong>
              </header>
              <div style={{ padding: '0 10px 10px' }}>
                <small>{t('browse.records_found')} <mark>{item.count}</mark></small>
              </div>
            </article>
          ))}
        </div>
      )}
      
      {results.length === 0 && !loading && !error && term && (
        <p style={{ textAlign: 'center', marginTop: '20px' }}>{t('browse.no_entries').replace('{term}', term)}</p>
      )}
    </>
  )
}
