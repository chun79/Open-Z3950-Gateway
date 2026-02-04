import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { SkeletonCard } from '../components/Skeletons'

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
  
  // New State for pagination
  const [scanPosition, setScanPosition] = useState(1) // Default: term is first
  
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

  const executeScan = async (scanTerm: string, position: number = 1, append: 'none' | 'top' | 'bottom' = 'none') => {
    setLoading(true)
    setError('')
    
    // If not appending, clear results immediately
    if (append === 'none') setResults([])

    try {
      const params = new URLSearchParams()
      params.append('term', scanTerm)
      params.append('db', targetDB)
      params.append('field', scanField)
      params.append('position', position.toString())
      params.append('count', '20')

      const response = await fetch(`/api/scan?${params.toString()}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      
      if (!response.ok) throw new Error(`Error: ${response.statusText}`)
      
      const data = await response.json()
      const list = data.data || []
      
      if (append === 'top') {
        // Remove the last item (which duplicates the scan term) if overlap occurs
        // Z39.50 scan logic can be tricky.
        // Simplified: Just prepend.
        setResults([...list, ...results])
      } else if (append === 'bottom') {
        // Z39.50 Scan usually returns the startTerm as first item if position=1.
        // So we might have a duplicate.
        const newList = list.length > 0 && list[0].term === scanTerm ? list.slice(1) : list
        setResults([...results, ...newList])
      } else {
        setResults(list)
      }
      
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setScanPosition(1) // Reset to standard forward scan
    executeScan(term, 1)
  }

  const handleAlphaClick = (letter: string) => {
    setTerm(letter)
    setScanPosition(1)
    executeScan(letter, 1)
  }

  const handleTermClick = (item: ScanResult) => {
    let attr = '4'
    if (scanField === 'author') attr = '1003'
    if (scanField === 'subject') attr = '21'
    navigate(`/?term=${encodeURIComponent(item.term)}&attr=${attr}&db=${targetDB}`)
  }

  const loadPrevious = () => {
    if (results.length === 0) return
    // To see what's before the first item, we scan that item but ask for it to be at position 21 (end of new list)
    // So we get 20 items BEFORE it.
    const firstTerm = results[0].term
    executeScan(firstTerm, 21, 'top')
  }

  const loadNext = () => {
    if (results.length === 0) return
    // Standard forward scan from last item
    const lastTerm = results[results.length - 1].term
    executeScan(lastTerm, 1, 'bottom')
  }

  return (
    <>
      <article>
        <header>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '10px' }}>
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
        <div style={{ marginTop: '20px' }}>
          <button 
            className="secondary outline" 
            onClick={loadPrevious}
            disabled={loading}
            style={{ width: '100%', marginBottom: '10px' }}
          >
            ⬆ Load Previous
          </button>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '0' }}>
            {results.map((item, index) => (
              <div 
                key={`${item.term}-${index}`} 
                onClick={() => handleTermClick(item)} 
                className="scan-row"
                style={{ 
                  padding: '12px', 
                  borderBottom: '1px solid #eee', 
                  cursor: 'pointer',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  backgroundColor: item.term === term ? '#f0f8ff' : 'transparent'
                }}
              >
                <span style={{ fontWeight: item.term === term ? 'bold' : 'normal' }}>
                  {item.term}
                </span>
                <span style={{ 
                  backgroundColor: '#e0e0e0', 
                  padding: '2px 8px', 
                  borderRadius: '10px', 
                  fontSize: '0.8em',
                  color: '#555'
                }}>
                  {item.count}
                </span>
              </div>
            ))}
          </div>

          <button 
            className="secondary outline" 
            onClick={loadNext}
            disabled={loading}
            style={{ width: '100%', marginTop: '10px' }}
          >
            ⬇ Load Next
          </button>
        </div>
      )}
      
      {loading && results.length === 0 && <SkeletonCard />}
      
      {results.length === 0 && !loading && !error && term && (
        <p style={{ textAlign: 'center', marginTop: '20px' }}>{t('browse.no_entries').replace('{term}', term)}</p>
      )}
    </>
  )
}