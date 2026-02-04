import React, { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Book } from '../types'
import { useAuth } from '../context/AuthContext'
import { SkeletonCard } from '../components/Skeletons'
import { useI18n } from '../context/I18nContext'
import toast from 'react-hot-toast'

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

interface EnhancedBook extends Book {
  source_target?: string
}

export default function Search() {
  const { t } = useI18n()
  const { token } = useAuth()
  const location = useLocation()
  
  const [isAdvanced, setIsAdvanced] = useState(false)
  const [isFederated, setIsFederated] = useState(false)
  const [simpleQuery, setSimpleQuery] = useState('')
  const [rows, setRows] = useState<QueryRow[]>([{ id: Date.now(), attribute: '1016', term: '', operator: 'AND' }])
  const [targets, setTargets] = useState<string[]>([])
  const [targetDB, setTargetDB] = useState('LCDB')
  const [selectedTargets, setSelectedTargets] = useState<string[]>([])
  const [results, setResults] = useState<EnhancedBook[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [history, setHistory] = useState<SearchHistoryItem[]>([])
  const [showHistory, setShowHistory] = useState(false)

  const transport = createConnectTransport({ baseUrl: window.location.origin })
  const client = createPromiseClient(GatewayService, transport)

  useEffect(() => {
    const saved = localStorage.getItem('z3950_search_history')
    if (saved) try { setHistory(JSON.parse(saved)) } catch (e) {}
    
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
  }, [token])

  const saveToHistory = (item: SearchHistoryItem) => {
    const newHistory = [item, ...history.filter(h => h.summary !== item.summary)].slice(0, 10)
    setHistory(newHistory)
    localStorage.setItem('z3950_search_history', JSON.stringify(newHistory))
  }

  const loadHistory = (item: SearchHistoryItem) => {
    setTargetDB(item.db)
    if (item.type === 'federated') {
      setIsFederated(true); setIsAdvanced(false)
      const tNames = item.db.split(',')
      setSelectedTargets(tNames); setSimpleQuery(item.query || '')
      doFederatedSearch(tNames, item.query || '')
    } else if (item.type === 'advanced' && item.rows) {
      setIsFederated(false); setIsAdvanced(true); setRows(item.rows)
      doSearch(item.db, item.rows)
    } else if (item.query) {
      setIsFederated(false); setIsAdvanced(false); setSimpleQuery(item.query)
      doSearch(item.db, null, item.query)
    }
    setShowHistory(false)
  }

  const doSearch = async (db: string, advancedRows?: any[], simpleTerm?: string) => {
    setLoading(true); setError(''); setResults([])
    try {
      const params = new URLSearchParams(); params.append('db', db)
      let summary = ""
      if (advancedRows) {
        advancedRows.forEach((row: any, idx: number) => {
          const i = idx + 1; params.append(`term${i}`, row.term); params.append(`attr${i}`, row.attribute)
          if (i > 1) params.append(`op${i}`, row.operator)
        })
        summary = `[${db}] ` + advancedRows.map((r:any) => r.term).join(" AND ")
      } else if (simpleTerm) {
        params.append('term1', simpleTerm); params.append('attr1', '1016')
        summary = `[${db}] ${simpleTerm}`
      }
      const response = await fetch(`/api/search?${params.toString()}`, { headers: { 'Authorization': `Bearer ${token}` } })
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || "Search failed")
      setResults(data.data || [])
      saveToHistory({ timestamp: Date.now(), db, type: advancedRows ? 'advanced' : 'simple', query: simpleTerm, rows: advancedRows, summary })
    } catch (err: any) { setError(err.message) } finally { setLoading(false) }
  }

  const doFederatedSearch = async (targetNames: string[], query: string) => {
    setLoading(true); setError(''); setResults([])
    try {
      const responses = client.search({ query, targets: targetNames, limit: 10 }, { headers: { "Authorization": `Bearer ${token}` } })
      for await (const res of responses) {
        if (res.result.case === "record") {
          const rec = res.result.value
          setResults(prev => [...prev, { record_id: rec.recordId, title: rec.title, author: rec.author, isbn: rec.isbn, publisher: rec.publisher, pub_year: rec.year, source_target: rec.sourceTarget }])
        }
      }
      saveToHistory({ timestamp: Date.now(), db: targetNames.join(','), type: 'federated', query, summary: `[Federated] ${query}` })
    } catch (err: any) { setError(err.message) } finally { setLoading(false) }
  }

  const handleIngest = async (book: EnhancedBook) => {
    const loadingToast = toast.loading(`Ingesting "${book.title}"...`)
    try {
      const source = book.source_target || targetDB
      const resDetail = await fetch(`/api/books/${source}/${encodeURIComponent(book.record_id || '')}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      const detailData = await resDetail.json()
      if (!resDetail.ok) throw new Error("Failed to fetch full record")

      const resIngest = await fetch('/api/books', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify(detailData.data)
      })
      const ingestData = await resIngest.json()
      if (!resIngest.ok) throw new Error(ingestData.error || "Ingest failed")

      toast.success(`Ingested! Local ID: ${ingestData.id}`, { id: loadingToast })
    } catch (err: any) {
      toast.error(err.message, { id: loadingToast })
    }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (isFederated) doFederatedSearch(selectedTargets, simpleQuery)
    else if (isAdvanced) doSearch(targetDB, rows)
    else doSearch(targetDB, null, simpleQuery)
  }

  const cleanISBN = (isbn: string) => isbn ? isbn.replace(/[^0-9X]/gi, '') : ''

  return (
    <>
      <article>
        <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div role="group" style={{ marginBottom: 0 }}>
            <button className={(!isAdvanced && !isFederated) ? "" : "outline"} onClick={() => {setIsAdvanced(false); setIsFederated(false)}}>Simple</button>
            <button className={isAdvanced ? "" : "outline"} onClick={() => {setIsAdvanced(true); setIsFederated(false)}}>Advanced</button>
            <button className={isFederated ? "" : "outline"} onClick={() => {setIsAdvanced(false); setIsFederated(true)}}>⚡ Stream</button>
          </div>
          {history.length > 0 && <button className="outline secondary" onClick={() => setShowHistory(!showHistory)} style={{padding: '5px 10px', fontSize: '0.8em', marginBottom: 0}}>⏱ History</button>}
        </header>

        {showHistory && (
          <div style={{ padding: '10px', background: '#f9f9f9', marginBottom: '20px', borderRadius: '4px', border: '1px solid #ddd' }}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '5px' }}>
              {history.map((item, idx) => <button key={idx} className="outline secondary" onClick={() => loadHistory(item)} style={{ padding: '5px 10px', fontSize: '0.8em', marginBottom: 0 }}>{item.summary}</button>)}
            </div>
          </div>
        )}

        <form onSubmit={handleSearch}>
          <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
            <input type="search" placeholder="Search global libraries..." value={simpleQuery} onChange={(e) => setSimpleQuery(e.target.value)} required style={{ flexGrow: 1, marginBottom: 0 }} />
            <button type="submit" disabled={loading}>{loading ? '...' : 'Search'}</button>
          </div>
        </form>
      </article>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
        {loading && results.length === 0 && [1, 2, 3, 4].map(i => <SkeletonCard key={i} />)}
        {results.map((item, index) => (
          <article key={index} style={{ marginBottom: 0 }}>
            <div style={{ display: 'flex', gap: '20px' }}>
              <Link to={`/book/${item.source_target || targetDB}/${encodeURIComponent(item.record_id || '')}`}>
                <img src={`https://covers.openlibrary.org/b/isbn/${cleanISBN(item.isbn || '')}-M.jpg?default=https://placehold.co/100x150/e0e0e0/808080?text=No+Cover`} style={{ width: '80px', borderRadius: '4px' }} />
              </Link>
              <div style={{ flex: 1 }}>
                <h6 style={{ marginBottom: '5px' }}>{item.title}</h6>
                <small style={{ display: 'block', marginBottom: '10px' }}>{item.author}</small>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <mark style={{ fontSize: '0.6em', padding: '2px 5px' }}>{item.source_target || targetDB}</mark>
                  <button className="outline" onClick={() => handleIngest(item)} style={{ padding: '2px 8px', fontSize: '0.7em', marginBottom: 0 }}>📥 Ingest</button>
                </div>
              </div>
            </div>
          </article>
        ))}
      </div>
    </>
  )
}
