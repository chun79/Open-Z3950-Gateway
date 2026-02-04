import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Book } from '../types'
import { useI18n } from '../context/I18nContext'
import { Search as WailsSearch, ListTargets, SaveBook, ListSearchHistory, ClearSearchHistory } from '../../wailsjs/go/main/App'
import { SkeletonCard } from '../components/Skeletons'

export default function Search() {
  const { t } = useI18n()
  
  const ATTRIBUTES = [
    { value: '1016', label: t('search.attr.any') },
    { value: '4', label: t('search.attr.title') },
    { value: '1003', label: t('search.attr.author') },
    { value: '7', label: t('search.attr.isbn') },
  ]

  const [term, setTerm] = useState('')
  const [attr, setAttr] = useState('1016')
  const [targets, setTargets] = useState<string[]>([])
  const [selectedDBs, setSelectedDBs] = useState<string[]>([])
  
  const [results, setResults] = useState<Book[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [history, setHistory] = useState<string[]>([])

  useEffect(() => {
    ListTargets().then(list => {
      const names = list.map((t: any) => t.name)
      setTargets(names)
      if (names.length > 0) setSelectedDBs([names[0]]) // Select first by default
    })
    loadHistory()
  }, [])

  const loadHistory = () => {
    ListSearchHistory().then(h => setHistory(h || []))
  }

  const handleClearHistory = async () => {
    await ClearSearchHistory()
    loadHistory()
  }

  const handleSearch = async (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    if (selectedDBs.length === 0) {
      setError("Please select at least one library")
      return
    }

    setLoading(true)
    setError('')
    setResults([])

    try {
      const data = await WailsSearch({
        dbs: selectedDBs,
        term: term,
        attr: parseInt(attr)
      })
      
      setResults(data || [])
      if (!data || data.length === 0) setError(t('search.no_results'))
      loadHistory()
    } catch (err: any) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  const handleHistoryClick = (h: string) => {
    setTerm(h)
  }

  const handleSave = async (book: Book) => {
    try {
      await SaveBook({
        title: book.title,
        author: book.author,
        isbn: book.isbn,
        source_db: (book as any).source_db || "Unknown"
      })
      alert("Book saved to local shelf!")
    } catch (err: any) {
      alert("Failed to save: " + err)
    }
  }

  const toggleDB = (db: string) => {
    if (selectedDBs.includes(db)) {
      setSelectedDBs(selectedDBs.filter(d => d !== db))
    } else {
      setSelectedDBs([...selectedDBs, db])
    }
  }

  return (
    <>
      <article>
        <header>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <strong>Broadcast Search</strong>
            <small style={{ fontSize: '0.7em', color: '#666' }}>
              Searching {selectedDBs.length} targets
            </small>
          </div>
        </header>
        <form onSubmit={handleSearch}>
          <div className="grid">
            <select 
              value={attr} 
              onChange={(e) => setAttr(e.target.value)}
              style={{ width: '150px' }}
            >
              {ATTRIBUTES.map(a => <option key={a.value} value={a.value}>{a.label}</option>)}
            </select>
            <input 
              type="search" 
              placeholder={t('search.placeholder')} 
              value={term}
              onChange={(e) => setTerm(e.target.value)}
              required
              style={{ flexGrow: 1 }}
            />
            <button type="submit" disabled={loading}>
              {loading ? '...' : t('search.button')}
            </button>
          </div>

          <details style={{ marginTop: '10px' }}>
            <summary style={{ fontSize: '0.9em', cursor: 'pointer' }}>
              Select Libraries <span style={{ opacity: 0.6 }}>({selectedDBs.join(', ')})</span>
            </summary>
            <div style={{ 
              display: 'grid', 
              gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', 
              gap: '10px', 
              padding: '15px', 
              background: 'var(--pico-card-background-color)', 
              border: '1px solid var(--pico-muted-border-color)', 
              borderRadius: '4px',
              marginTop: '10px'
            }}>
              {targets.map(t => (
                <label key={t} style={{ fontSize: '0.9em', cursor: 'pointer' }}>
                  <input 
                    type="checkbox" 
                    checked={selectedDBs.includes(t)} 
                    onChange={() => toggleDB(t)} 
                  />
                  {t}
                </label>
              ))}
            </div>
          </details>
        </form>
        
        {history.length > 0 && !loading && results.length === 0 && !error && (
          <div style={{ marginTop: '15px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <small>Recent Searches:</small>
              <button 
                className="outline secondary" 
                style={{ border: 'none', padding: '0 5px', fontSize: '0.7em', height: 'auto', marginBottom: 0 }}
                onClick={handleClearHistory}
              >
                Clear
              </button>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginTop: '5px' }}>
              {history.map((h, idx) => (
                <button 
                  key={idx} 
                  className="outline secondary" 
                  style={{ 
                    padding: '2px 10px', 
                    fontSize: '0.8em', 
                    borderRadius: '12px',
                    border: '1px solid var(--pico-secondary-border)' 
                  }}
                  onClick={() => handleHistoryClick(h)}
                >
                  {h}
                </button>
              ))}
            </div>
          </div>
        )}
      </article>

      {error && <article className="pico-background-red-200">❌ {error}</article>}

      {loading ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
          {[1, 2, 3, 4].map(i => <SkeletonCard key={i} />)}
        </div>
      ) : results.length > 0 ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
          {results.map((item, index) => (
            <article key={index} style={{ position: 'relative' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '10px' }}>
                <Link to={`/book/${(item as any).source_db}/${item.isbn || item.record_id}`} style={{ textDecoration: 'none', color: 'inherit', flexGrow: 1 }}>
                  <strong>{item.title || t('common.untitled')}</strong>
                </Link>
                <div style={{ display: 'flex', gap: '5px', alignItems: 'center' }}>
                  {(item as any).source_db && (
                    <mark style={{ fontSize: '0.6em', padding: '2px 6px', borderRadius: '4px', whiteSpace: 'nowrap' }}>
                      {(item as any).source_db}
                    </mark>
                  )}
                  <button 
                    className="outline secondary" 
                    style={{ padding: '2px 6px', fontSize: '0.8em', border: 'none', marginBottom: 0 }}
                    onClick={() => handleSave(item)}
                    title="Save to Bookshelf"
                  >
                    ⭐
                  </button>
                </div>
              </div>
              
              <div style={{ fontSize: '0.9em' }}>
                <p style={{ marginBottom: '5px' }}><strong>{t('search.attr.author')}:</strong> {item.author || t('common.unknown')}</p>
                <p style={{ marginBottom: '5px' }}><strong>ISBN:</strong> {item.isbn}</p>
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </>
  )
}