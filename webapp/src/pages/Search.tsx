import React, { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Book } from '../types'
import { useAuth } from '../context/AuthContext'
import { generateBibTeX, generateRIS } from '../utils/citation'
import { SkeletonCard } from '../components/Skeletons'
import { useI18n } from '../context/I18nContext'

// ConnectRPC Imports
import { createPromiseClient } from "@connectrpc/connect"
import { createConnectTransport } from "@connectrpc/connect-web"
import { GatewayService } from "../gen/proto/gateway/v1/gateway_connect"

type QueryRow = {
  id: number
  attribute: string
  term: string
  operator: string
}

type SearchHistoryItem = {
  timestamp: number
  db: string
  type: 'simple' | 'advanced' | 'federated'
  query?: string
  rows?: QueryRow[]
  summary: string
}

// Extended Book type to include source_target from federated search
interface EnhancedBook extends Book {
  source_target?: string
}

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
  const [isFederated, setIsFederated] = useState(false)
  const [simpleQuery, setSimpleQuery] = useState('')
  
  const [rows, setRows] = useState<QueryRow[]>([
    { id: Date.now(), attribute: '1016', term: '', operator: 'AND' }
  ])
  
  const [targets, setTargets] = useState<string[]>([])
  const [targetDB, setTargetDB] = useState('LCDB')
  const [selectedTargets, setSelectedTargets] = useState<string[]>([])
  
  const [results, setResults] = useState<EnhancedBook[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [requestStatus, setRequestStatus] = useState<{msg: string, type: 'success' | 'error'} | null>(null)
  
  const [history, setHistory] = useState<SearchHistoryItem[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [citation, setCitation] = useState<{ content: string, format: string } | null>(null)

  const { token } = useAuth()
  const location = useLocation()

  // ConnectRPC Setup
  const transport = createConnectTransport({
    baseUrl: window.location.origin, // Use current origin
  })
  const client = createPromiseClient(GatewayService, transport)

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
    if (item.type === 'federated') {
      setIsFederated(true)
      setIsAdvanced(false)
      const tNames = item.db.split(',')
      setSelectedTargets(tNames)
      setSimpleQuery(item.query || '')
      doFederatedSearch(tNames, item.query || '')
    } else if (item.type === 'advanced' && item.rows) {
      setIsFederated(false)
      setIsAdvanced(true)
      setRows(item.rows)
      doSearch(item.db, item.rows)
    } else if (item.query) {
      setIsFederated(false)
      setIsAdvanced(false)
      setSimpleQuery(item.query)
      doSearch(item.db, null, item.query)
    }
    setShowHistory(false)
  }

  useEffect(() => {
    fetch('/api/targets', { headers: { 'Authorization': `Bearer ${token}` } })
      .then(res => res.json())
      .then(data => {
        if (data.status === 'success' && data.data) {
          setTargets(data.data)
          setSelectedTargets(data.data.slice(0, 2))
          if (!new URLSearchParams(location.search).get('db') && data.data.length > 0) {
            setTargetDB(data.data[0])
          }
        }
      })
      .catch(console.error)
  }, [token])

  // Simple search using standard REST API
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
        summary = `[${db}] ` + advancedRows.map(r => `${r.term}`).join(" AND ")
      } else if (simpleTerm) {
        params.append('term1', simpleTerm)
        params.append('attr1', '1016')
        summary = `[${db}] ${simpleTerm}`
      }

      const response = await fetch(`/api/search?${params.toString()}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      
      if (!response.ok) throw new Error(`Error: ${response.statusText}`)
      const data = await response.json()
      const list = data.data || []
      setResults(list)
      if (list.length === 0) setError(t('search.no_results'))
        
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

  // --- Federated search using ConnectRPC Streaming ---
  const doFederatedSearch = async (targetNames: string[], query: string) => {
    setLoading(true)
    setError('')
    setResults([])
    setRequestStatus(null)

    try {
      // ConnectRPC Streaming Call
      const responses = client.search({
        query: query,
        targets: targetNames,
        limit: 10
      }, {
        headers: { "Authorization": `Bearer ${token}` }
      })

      let count = 0
      for await (const res of responses) {
        if (res.result.case === "record") {
          const rec = res.result.value
          // Update state incrementally!
          setResults(prev => [...prev, {
            record_id: rec.recordId,
            title: rec.title,
            author: rec.author,
            isbn: rec.isbn,
            publisher: rec.publisher,
            pub_year: rec.year,
            source_target: rec.sourceTarget
          }])
          count++
        } else if (res.result.case === "status") {
          const status = res.result.value
          console.log(`Target ${status.target}: ${status.message}`)
        }
      }

      if (count === 0) setError(t('search.no_results'))

      saveToHistory({
        timestamp: Date.now(),
        db: targetNames.join(','),
        type: 'federated',
        query: query,
        summary: `[Federated: ${targetNames.length}] ${query}`
      })
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isFederated) {
      doFederatedSearch(selectedTargets, simpleQuery)
    } else if (isAdvanced) {
      doSearch(targetDB, rows)
    } else {
      doSearch(targetDB, null, simpleQuery)
    }
  }

  const toggleTarget = (name: string) => {
    if (selectedTargets.includes(name)) {
      setSelectedTargets(selectedTargets.filter(t => t !== name))
    } else {
      setSelectedTargets([...selectedTargets, name])
    }
  }

  const cleanISBN = (isbn: string) => {
    return isbn.replace(/[^0-9X]/gi, '')
  }

  return (
    <>
      <article>
        <header>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '10px' }}>
            <div role="group" style={{ marginBottom: 0 }}>
              <button 
                className={(!isAdvanced && !isFederated) ? "" : "outline"} 
                onClick={() => { setIsAdvanced(false); setIsFederated(false); }}
                style={{fontSize: '0.9em', padding: '5px 15px'}}
              >
                {t('search.simple')}
              </button>
              <button 
                className={isAdvanced ? "" : "outline"} 
                onClick={() => { setIsAdvanced(true); setIsFederated(false); }}
                style={{fontSize: '0.9em', padding: '5px 15px'}}
              >
                {t('search.advanced')}
              </button>
              <button 
                className={isFederated ? "" : "outline"} 
                onClick={() => { setIsAdvanced(false); setIsFederated(true); }}
                style={{fontSize: '0.9em', padding: '5px 15px'}}
              >
                ⚡ Streaming
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
              
              {!isFederated && (
                <div style={{display: 'flex', flexDirection: 'column', gap: '2px'}}>
                  <small style={{fontSize: '0.7em'}}>{t('search.target')}</small>
                  <select 
                    value={targetDB} 
                    onChange={(e) => setTargetDB(e.target.value)}
                    style={{ width: 'auto', marginBottom: 0, padding: '5px', fontSize: '0.8em', height: 'auto' }}
                  >
                    {targets.map(t => <option key={t} value={t}>{t}</option>)}
                    {targets.length === 0 && <option value="LCDB">LCDB</option>}
                  </select>
                </div>
              )}
            </div>
          </div>
        </header>

        {showHistory && history.length > 0 && (
          <div style={{ padding: '10px', background: '#f9f9f9', marginBottom: '20px', borderRadius: '4px', border: '1px solid #ddd' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
              <strong>Recent Searches</strong>
              <button className="outline contrast" onClick={() => {setHistory([]); localStorage.removeItem('z3950_search_history')}} style={{ padding: '2px 8px', fontSize: '0.7em', width: 'auto', marginBottom: 0 }}>Clear</button>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '5px' }}>
              {history.map((item, idx) => (
                <button key={idx} className="outline secondary" onClick={() => loadHistory(item)} style={{ padding: '5px 10px', fontSize: '0.8em', marginBottom: 0 }}>
                  {item.summary}
                </button>
              ))}
            </div>
          </div>
        )}

        {isFederated && (
          <div style={{ marginBottom: '20px', padding: '15px', border: '1px dashed #ccc', borderRadius: '8px' }}>
            <small style={{ display: 'block', marginBottom: '10px', fontWeight: 'bold' }}>Streaming Federated Search Targets:</small>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '15px' }}>
              {targets.map(t => (
                <label key={t} style={{ display: 'flex', alignItems: 'center', gap: '5px', fontSize: '0.9em', cursor: 'pointer' }}>
                  <input type="checkbox" checked={selectedTargets.includes(t)} onChange={() => toggleTarget(t)} style={{ marginBottom: 0 }} />
                  {t}
                </label>
              ))}
            </div>
          </div>
        )}

        <form onSubmit={handleSearch}>
          {(!isAdvanced) ? (
            <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
              <input 
                type="search" 
                placeholder={isFederated ? "Streaming search across all targets..." : t('search.placeholder')} 
                value={simpleQuery}
                onChange={(e) => setSimpleQuery(e.target.value)}
                required
                style={{ flexGrow: 1, marginBottom: 0, fontSize: '1.1em', padding: '12px' }}
              />
              <button type="submit" disabled={loading} style={{fontSize: '1.1em', padding: '12px 30px'}}>
                {loading ? 'Searching...' : t('search.button')}
              </button>
            </div>
          ) : (
            <>
              {/* Advanced RPN UI remains same, calling doSearch */}
              {rows.map((row, index) => (
                <div key={row.id} style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
                  <input type="text" value={row.term} onChange={(e) => {
                    const newRows = [...rows]; newRows[index].term = e.target.value; setRows(newRows);
                  }} required style={{ flexGrow: 1, marginBottom: 0 }} />
                </div>
              ))}
              <button type="submit" disabled={loading}>{t('search.button')}</button>
            </>
          )}
        </form>
      </article>

      {error && <article className="pico-background-red-200">❌ {error}</article>}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
        {loading && results.length === 0 && [1, 2, 3, 4].map(i => <SkeletonCard key={i} />)}
        {results.map((item, index) => (
          <article key={index}>
            <div style={{ display: 'flex', gap: '20px' }}>
              <Link to={`/book/${item.source_target || targetDB}/${encodeURIComponent(item.record_id || '')}`}>
                <img 
                  src={item.isbn ? `https://covers.openlibrary.org/b/isbn/${cleanISBN(item.isbn)}-M.jpg?default=https://placehold.co/100x150/e0e0e0/808080?text=No+Cover` : 'https://placehold.co/100x150/e0e0e0/808080?text=No+ISBN'} 
                  style={{ width: '100px', borderRadius: '4px' }} 
                />
              </Link>
              <div>
                <strong>{item.title}</strong>
                <p style={{fontSize: '0.9em', margin: '5px 0'}}>{item.author}</p>
                {item.source_target && <mark style={{fontSize: '0.7em', padding: '2px 5px'}}>{item.source_target}</mark>}
              </div>
            </div>
          </article>
        ))}
      </div>
    </>
  )
}