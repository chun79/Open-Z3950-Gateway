import React, { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Book } from '../types'
import { useAuth } from '../context/AuthContext'
import { generateBibTeX, generateRIS } from '../utils/citation'
import { SkeletonCard } from '../components/Skeletons'
import { useI18n } from '../context/I18nContext'

type QueryRow = {
  id: number
  attribute: string
  term: string
  operator: string
}

type SearchHistoryItem = {
  timestamp: number
  db: string
  type: 'simple' | 'advanced'
  query?: string
  rows?: QueryRow[]
  summary: string
}

const OPERATORS = [
  { value: 'AND', label: 'AND' },
  { value: 'OR', label: 'OR' },
  { value: 'AND-NOT', label: 'NOT' },
]

export default function Search() {
  const { t } = useI18n()
  
  const ATTRIBUTES = [
    { value: '1016', label: t('search.attr.any') },
    { value: '4', label: t('search.attr.title') },
    { value: '1003', label: t('search.attr.author') },
    { value: '7', label: t('search.attr.isbn') },
    { value: '8', label: t('search.attr.issn') },
    { value: '21', label: t('search.attr.subject') },
    { value: '31', label: t('search.attr.date') },
  ]

  const [isAdvanced, setIsAdvanced] = useState(false)
  const [simpleQuery, setSimpleQuery] = useState('')
  
  const [rows, setRows] = useState<QueryRow[]>([
    { id: Date.now(), attribute: '1016', term: '', operator: 'AND' }
  ])
  
  const [targets, setTargets] = useState<string[]>([])
  const [targetDB, setTargetDB] = useState('LCDB')
  
  const [results, setResults] = useState<Book[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [requestStatus, setRequestStatus] = useState<{msg: string, type: 'success' | 'error'} | null>(null)
  
  // History State
  const [history, setHistory] = useState<SearchHistoryItem[]>([])
  const [showHistory, setShowHistory] = useState(false)

  // Citation Modal State
  const [citation, setCitation] = useState<{ content: string, format: string } | null>(null)

  const { token } = useAuth()
  const location = useLocation()

  useEffect(() => {
    const saved = localStorage.getItem('z3950_search_history')
    if (saved) {
      try {
        setHistory(JSON.parse(saved))
      } catch (e) {
        console.error("Failed to parse history", e)
      }
    }
  }, [])

  const saveToHistory = (item: SearchHistoryItem) => {
    const newHistory = [item, ...history.filter(h => h.summary !== item.summary)].slice(0, 10)
    setHistory(newHistory)
    localStorage.setItem('z3950_search_history', JSON.stringify(newHistory))
  }

  const loadHistory = (item: SearchHistoryItem) => {
    setTargetDB(item.db)
    if (item.type === 'advanced' && item.rows) {
      setIsAdvanced(true)
      setRows(item.rows)
      doSearch(item.db, item.rows)
    } else if (item.query) {
      setIsAdvanced(false)
      setSimpleQuery(item.query)
      doSearch(item.db, null, item.query)
    }
    setShowHistory(false)
  }

  const clearHistory = () => {
    setHistory([])
    localStorage.removeItem('z3950_search_history')
  }

  useEffect(() => {
    fetch('/api/targets', { headers: { 'Authorization': `Bearer ${token}` } })
      .then(res => res.json())
      .then(data => {
        if (data.status === 'success' && data.data) {
          setTargets(data.data)
          // Only set default if no query param
          if (!new URLSearchParams(location.search).get('db') && data.data.length > 0) {
            setTargetDB(data.data[0])
          }
        }
      })
      .catch(console.error)
  }, [token])

  // Handle query params from Browse page
  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const term = params.get('term')
    const attr = params.get('attr')
    const db = params.get('db')

    if (db) setTargetDB(db)

    if (term) {
      if (attr && attr !== '1016') {
        // Switch to Advanced for specific attribute search
        setIsAdvanced(true)
        setRows([{ id: Date.now(), attribute: attr, term: term, operator: 'AND' }])
        // Trigger search immediately
        doSearch(db || 'LCDB', [{ attribute: attr, term: term, operator: 'AND' }])
      } else {
        // Simple search
        setIsAdvanced(false)
        setSimpleQuery(term)
        doSearch(db || 'LCDB', null, term)
      }
    }
  }, [location.search, token])

  const doSearch = async (db: string, advancedRows?: any[], simpleTerm?: string) => {
    setLoading(true)
    setError('')
    setResults([])
    setRequestStatus(null)

    try {
      const params = new URLSearchParams()
      params.append('db', db)
      
      let summary = ""

      if (advancedRows) {
        advancedRows.forEach((row, index) => {
          const i = index + 1
          params.append(`term${i}`, row.term)
          params.append(`attr${i}`, row.attribute)
          if (i > 1) params.append(`op${i}`, row.operator)
        })
        summary = `[${db}] ` + advancedRows.map(r => `${r.term} (${ATTRIBUTES.find(a => a.value === r.attribute)?.label || r.attribute})`).join(" AND ")
      } else if (simpleTerm) {
        params.append('term1', simpleTerm)
        params.append('attr1', '1016')
        summary = `[${db}] ${simpleTerm}`
      } else {
        setLoading(false)
        return // Nothing to search
      }

      const response = await fetch(`/api/search?${params.toString()}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      
      if (!response.ok) throw new Error(`Error: ${response.statusText}`)
      const data = await response.json()
      if (data.error) throw new Error(data.error)
      
      const list = data.data || []
      setResults(list)
      if (list.length === 0) setError(t('search.no_results'))
        
      // Save history on success (even if 0 results, valid query)
      saveToHistory({
        timestamp: Date.now(),
        db,
        type: advancedRows ? 'advanced' : 'simple',
        query: simpleTerm,
        rows: advancedRows,
        summary
      })

    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const addRow = () => {
    setRows([...rows, { id: Date.now(), attribute: '1016', term: '', operator: 'AND' }])
  }

  const removeRow = (index: number) => {
    if (rows.length > 1) {
      setRows(rows.filter((_, i) => i !== index))
    }
  }

  const updateRow = (index: number, field: keyof QueryRow, value: string) => {
    const newRows = [...rows]
    newRows[index] = { ...newRows[index], [field]: value }
    setRows(newRows)
  }

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isAdvanced) {
      doSearch(targetDB, rows)
    } else {
      doSearch(targetDB, null, simpleQuery)
    }
  }

  const handleILLRequest = async (book: Book) => {
    try {
      const response = await fetch('/api/ill-requests', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          title: book.title,
          author: book.author,
          isbn: book.isbn,
          target_db: targetDB,
          record_id: book.record_id || 'unknown'
        })
      })

      if (!response.ok) {
        const errData = await response.json()
        throw new Error(errData.error || 'Request failed')
      }

      setRequestStatus({ msg: t('detail.request_success', {title: book.title}), type: 'success' })
    } catch (err: any) {
      setRequestStatus({ msg: t('detail.request_fail', {error: err.message}), type: 'error' })
    }
  }

  const showCite = (book: Book, format: 'BibTeX' | 'RIS') => {
    const content = format === 'BibTeX' ? generateBibTeX(book) : generateRIS(book)
    setCitation({ content, format })
  }

  const cleanISBN = (isbn: string) => {
    return isbn.replace(/[^0-9X]/gi, '')
  }

  return (
    <>
      <article>
        <header>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div role="group">
              <button 
                className={!isAdvanced ? "" : "outline"} 
                onClick={() => setIsAdvanced(false)}
                style={{marginBottom: 0, fontSize: '0.9em', padding: '5px 15px'}}
              >
                {t('search.simple')}
              </button>
              <button 
                className={isAdvanced ? "" : "outline"} 
                onClick={() => setIsAdvanced(true)}
                style={{marginBottom: 0, fontSize: '0.9em', padding: '5px 15px'}}
              >
                {t('search.advanced')}
              </button>
            </div>
            
            <div style={{display: 'flex', alignItems: 'center', gap: '10px'}}>
              {history.length > 0 && (
                <button 
                  className="outline secondary" 
                  onClick={() => setShowHistory(!showHistory)}
                  style={{padding: '5px 10px', fontSize: '0.8em', marginBottom: 0}}
                >
                  ⏱ History
                </button>
              )}
              <small>{t('search.target')}</small>
              <select 
                value={targetDB} 
                onChange={(e) => setTargetDB(e.target.value)}
                style={{ width: 'auto', marginBottom: 0 }}
              >
                {targets.map(t => <option key={t} value={t}>{t}</option>)}
                {targets.length === 0 && <option value="LCDB">LCDB</option>}
              </select>
            </div>
          </div>
        </header>

        {showHistory && history.length > 0 && (
          <div style={{ padding: '10px', background: '#f9f9f9', marginBottom: '20px', borderRadius: '4px', border: '1px solid #ddd' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
              <strong>Recent Searches</strong>
              <button className="outline contrast" onClick={clearHistory} style={{ padding: '2px 8px', fontSize: '0.7em', width: 'auto', marginBottom: 0 }}>Clear</button>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '5px' }}>
              {history.map((item, idx) => (
                <button 
                  key={idx} 
                  className="outline secondary" 
                  onClick={() => loadHistory(item)}
                  style={{ padding: '5px 10px', fontSize: '0.8em', marginBottom: 0 }}
                >
                  {item.summary}
                </button>
              ))}
            </div>
          </div>
        )}

        <form onSubmit={handleSearch}>
          {!isAdvanced ? (
            <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
              <input 
                type="search" 
                placeholder={t('search.placeholder')} 
                value={simpleQuery}
                onChange={(e) => setSimpleQuery(e.target.value)}
                required
                style={{ flexGrow: 1, marginBottom: 0, fontSize: '1.1em', padding: '12px' }}
              />
              <button type="submit" disabled={loading} style={{fontSize: '1.1em', padding: '12px 30px'}}>
                {loading ? '...' : t('search.button')}
              </button>
            </div>
          ) : (
            <>
              {rows.map((row, index) => (
                <div key={row.id} style={{ display: 'flex', gap: '10px', marginBottom: '10px', alignItems: 'center' }}>
                  {index > 0 && (
                    <select 
                      value={row.operator} 
                      onChange={(e) => updateRow(index, 'operator', e.target.value)}
                      style={{ width: '100px', marginBottom: 0 }}
                    >
                      {OPERATORS.map(op => <option key={op.value} value={op.value}>{op.label}</option>)}
                    </select>
                  )}
                  
                  <select 
                    value={row.attribute} 
                    onChange={(e) => updateRow(index, 'attribute', e.target.value)}
                    style={{ width: '150px', marginBottom: 0 }}
                  >
                    {ATTRIBUTES.map(attr => <option key={attr.value} value={attr.value}>{attr.label}</option>)}
                  </select>

                  <input 
                    type="text" 
                    placeholder={t('search.term_placeholder')}
                    value={row.term}
                    onChange={(e) => updateRow(index, 'term', e.target.value)}
                    required
                    style={{ flexGrow: 1, marginBottom: 0 }}
                  />

                  {rows.length > 1 && (
                    <button 
                      type="button" 
                      className="secondary outline" 
                      onClick={() => removeRow(index)}
                      style={{ width: 'auto', marginBottom: 0, padding: '10px' }}
                    >
                      ✕
                    </button>
                  )}
                </div>
              ))}

              <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
                <button type="button" className="secondary" onClick={addRow}>
                  +
                </button>
                <button type="submit" disabled={loading}>
                  {loading ? '...' : t('search.button')}
                </button>
              </div>
            </>
          )}
        </form>
      </article>

      {error && (
        <article className="pico-background-red-200">
          <strong>{t('search.error')}:</strong> {error}
        </article>
      )}

      {requestStatus && (
        <article className={requestStatus.type === 'success' ? "pico-background-green-200" : "pico-background-red-200"}>
          <strong>{requestStatus.type === 'success' ? '✅' : '❌'}</strong> {requestStatus.msg}
        </article>
      )}

      {loading ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
          {[1, 2, 3, 4].map(i => <SkeletonCard key={i} />)}
        </div>
      ) : results.length > 0 ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
          {results.map((item, index) => (
            <article key={index}>
              <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
                <div style={{ flexShrink: 0 }}>
                  <Link to={`/book/${targetDB}/${encodeURIComponent(item.record_id || '')}`}>
                    <img 
                      src={item.isbn 
                        ? `https://covers.openlibrary.org/b/isbn/${cleanISBN(item.isbn)}-M.jpg?default=https://placehold.co/100x150/e0e0e0/808080?text=No+Cover`
                        : 'https://placehold.co/100x150/e0e0e0/808080?text=No+ISBN'
                      }
                      alt="Book Cover"
                      style={{ 
                        width: '100px', 
                        height: '150px', 
                        objectFit: 'cover',
                        borderRadius: '4px',
                        border: '1px solid #ddd'
                      }}
                      loading="lazy"
                    />
                  </Link>
                </div>
                
                <div style={{ flexGrow: 1 }}>
                  <header style={{ marginBottom: '10px' }}>
                    <strong>
                      <Link to={`/book/${targetDB}/${encodeURIComponent(item.record_id || '')}`} style={{textDecoration: 'none', color: 'inherit'}}>
                        {item.title || 'Untitled'}
                      </Link>
                    </strong>
                  </header>
                  <p style={{ marginBottom: '5px' }}><strong>Author:</strong> {item.author || 'Unknown'}</p>
                  <p style={{ marginBottom: '5px' }}><strong>ISBN:</strong> {item.isbn || 'Unknown'}</p>
                  {item.issn && <p style={{ marginBottom: '5px' }}><strong>ISSN:</strong> {item.issn}</p>}
                  {item.subject && <p style={{ marginBottom: '5px' }}><strong>Subject:</strong> {item.subject}</p>}
                  {item.publisher && <p style={{ marginBottom: '5px' }}><strong>Publisher:</strong> {item.publisher}</p>}
                  {item.pub_year && <p style={{ marginBottom: '5px' }}><strong>Year:</strong> {item.pub_year}</p>}

                  {item.holdings && item.holdings.length > 0 && (
                    <div style={{ marginTop: '10px', borderTop: '1px solid #eee', paddingTop: '10px' }}>
                      <small><strong>{t('search.result.holdings')}:</strong></small>
                      <table style={{ fontSize: '0.85em', marginBottom: 0 }}>
                        <thead>
                          <tr>
                            <th>{t('search.result.location')}</th>
                            <th>{t('search.result.call_number')}</th>
                            <th>{t('search.result.status')}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {item.holdings.map((h, i) => (
                            <tr key={i}>
                              <td>{h.location}</td>
                              <td>{h.call_number}</td>
                              <td>
                                <span style={{ 
                                  color: h.status === 'Available' ? 'green' : 'red',
                                  fontWeight: 'bold'
                                }}>
                                  {h.status}
                                </span>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              </div>
              
              <footer style={{ marginTop: '15px' }}>
                <div role="group" style={{ marginBottom: 0 }}>
                  <button onClick={() => handleILLRequest(item)}>
                    {t('search.action.request')}
                  </button>
                  <button className="secondary outline" onClick={() => showCite(item, 'BibTeX')}>
                    {t('search.action.bibtex')}
                  </button>
                  <button className="secondary outline" onClick={() => showCite(item, 'RIS')}>
                    {t('search.action.ris')}
                  </button>
                </div>
              </footer>
            </article>
          ))}
        </div>
      ) : null}

      {/* Citation Modal */}
      {citation && (
        <dialog open>
          <article>
            <header>
              <button aria-label="Close" rel="prev" onClick={() => setCitation(null)}></button>
              <strong>{t('search.citation.title').replace('{format}', citation.format)}</strong>
            </header>
            <pre style={{ backgroundColor: '#f4f4f4', padding: '10px', borderRadius: '5px' }}>
              {citation.content}
            </pre>
            <footer>
              <button 
                onClick={() => {
                  navigator.clipboard.writeText(citation.content)
                  setCitation(null)
                }}
              >
                {t('search.action.copy')}
              </button>
            </footer>
          </article>
        </dialog>
      )}
    </>
  )
}