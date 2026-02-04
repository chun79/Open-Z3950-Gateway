import React, { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Book } from '../types'
import { useAuth } from '../context/AuthContext'
import { SkeletonCard } from '../components/Skeletons'
import { useI18n } from '../context/I18nContext'
import toast from 'react-hot-toast'
import ISBNScanner from '../components/ISBNScanner'

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
  
  // Mobile Scan State
  const [showScanner, setShowScanner] = useState(false)

  const transport = createConnectTransport({ baseUrl: window.location.origin })
  const client = createPromiseClient(GatewayService, transport)

  useEffect(() => {
    // Check if URL has query params (from scan or direct link)
    const params = new URLSearchParams(location.search)
    const q = params.get('query')
    const attr = params.get('attr1')
    if (q) {
      setSimpleQuery(q)
      doSearch(targetDB, null, q) // Trigger search automatically
    }

    const saved = localStorage.getItem('z3950_search_history')
    if (saved) try { setHistory(JSON.parse(saved)) } catch (e) {}
    
    fetch('/api/targets', { headers: { 'Authorization': `Bearer ${token}` } })
      .then(res => res.json())
      .then(data => {
        if (data.status === 'success' && data.data) {
          setTargets(data.data)
          setSelectedTargets(data.data.slice(0, 2))
          if (!q && data.data.length > 0) {
            setTargetDB(data.data[0])
          }
        }
      })
  }, [token, location.search])

  const doSearch = async (db: string, advancedRows?: any[], simpleTerm?: string) => {
    setLoading(true); setError(''); setResults([])
    try {
      const params = new URLSearchParams(); params.append('db', db)
      if (advancedRows) {
        advancedRows.forEach((row: any, idx: number) => {
          const i = idx + 1; params.append(`term${i}`, row.term); params.append(`attr${i}`, row.attribute)
          if (i > 1) params.append(`op${i}`, row.operator)
        })
      } else if (simpleTerm) {
        params.append('term1', simpleTerm); params.append('attr1', '1016')
      }
      const response = await fetch(`/api/search?${params.toString()}`, { headers: { 'Authorization': `Bearer ${token}` } })
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || "Search failed")
      setResults(data.data || [])
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
      if (resIngest.ok) toast.success(`Ingested to local library!`, { id: loadingToast })
      else throw new Error("Ingest failed")
    } catch (err: any) { toast.error(err.message, { id: loadingToast }) }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (isFederated) doFederatedSearch(selectedTargets, simpleQuery)
    else doSearch(targetDB, null, simpleQuery)
  }

  return (
    <>
      <article>
        <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div role="group" style={{ marginBottom: 0 }}>
            <button className={(!isAdvanced && !isFederated) ? "" : "outline"} onClick={() => {setIsAdvanced(false); setIsFederated(false)}}>Simple</button>
            <button className={isFederated ? "" : "outline"} onClick={() => {setIsAdvanced(false); setIsFederated(true)}}>⚡ Stream</button>
          </div>
          <button className="outline secondary" onClick={() => setShowScanner(true)} style={{padding: '5px 15px', marginBottom: 0, fontSize: '1.2em'}}>📷</button>
        </header>

        {showScanner && <ISBNScanner onClose={() => setShowScanner(false)} />}

        <form onSubmit={handleSearch}>
          <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
            <input type="search" placeholder="Search title, author or scan ISBN..." value={simpleQuery} onChange={(e) => setSimpleQuery(e.target.value)} required style={{ flexGrow: 1, marginBottom: 0 }} />
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
                <img src={`https://covers.openlibrary.org/b/isbn/${item.isbn?.replace(/[^0-9X]/gi, '')}-M.jpg?default=https://placehold.co/100x150/e0e0e0/808080?text=No+Cover`} style={{ width: '80px', borderRadius: '4px' }} />
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