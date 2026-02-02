import React, { useState, useEffect } from 'react'
import { Book } from '../types'
import { useAuth } from '../context/AuthContext'
import { generateBibTeX, generateRIS } from '../utils/citation'

type QueryRow = {
  id: number
  attribute: string
  term: string
  operator: string
}

const ATTRIBUTES = [
  { value: '1016', label: 'Any' },
  { value: '4', label: 'Title' },
  { value: '1003', label: 'Author' },
  { value: '7', label: 'ISBN' },
  { value: '8', label: 'ISSN' },
  { value: '21', label: 'Subject' },
  { value: '31', label: 'Date' },
]

const OPERATORS = [
  { value: 'AND', label: 'AND' },
  { value: 'OR', label: 'OR' },
  { value: 'AND-NOT', label: 'NOT' },
]

export default function Search() {
  const [rows, setRows] = useState<QueryRow[]>([
    { id: Date.now(), attribute: '1016', term: '', operator: 'AND' }
  ])
  
  const [targets, setTargets] = useState<string[]>([])
  const [targetDB, setTargetDB] = useState('LCDB')
  
  const [results, setResults] = useState<Book[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [requestStatus, setRequestStatus] = useState<{msg: string, type: 'success' | 'error'} | null>(null)
  
  // Citation Modal State
  const [citation, setCitation] = useState<{ content: string, format: string } | null>(null)

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
    setLoading(true)
    setError('')
    setResults([])
    setRequestStatus(null)

    try {
      const params = new URLSearchParams()
      params.append('db', targetDB)
      
      rows.forEach((row, index) => {
        const i = index + 1
        params.append(`term${i}`, row.term)
        params.append(`attr${i}`, row.attribute)
        if (i > 1) {
          params.append(`op${i}`, row.operator)
        }
      })

      const response = await fetch(`/api/search?${params.toString()}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      
      if (!response.ok) {
        throw new Error(`Error: ${response.statusText}`)
      }
      
      const data = await response.json()
      if (data.error) throw new Error(data.error)
      
      const list = data.data || []
      setResults(list)
      if (list.length === 0) {
        setError('No results found.')
      }
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
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

      setRequestStatus({ msg: `Requested "${book.title}" from ${targetDB} successfully!`, type: 'success' })
    } catch (err: any) {
      setRequestStatus({ msg: `Failed to request: ${err.message}`, type: 'error' })
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
            <strong>Advanced Search</strong>
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

        <form onSubmit={handleSearch}>
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
                placeholder="Search term..." 
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
              + Add Condition
            </button>
            <button type="submit" disabled={loading}>
              {loading ? 'Searching...' : 'Search'}
            </button>
          </div>
        </form>
      </article>

      {error && (
        <article className="pico-background-red-200">
          <strong>Message:</strong> {error}
        </article>
      )}

      {requestStatus && (
        <article className={requestStatus.type === 'success' ? "pico-background-green-200" : "pico-background-red-200"}>
          <strong>{requestStatus.type === 'success' ? '✅' : '❌'}</strong> {requestStatus.msg}
        </article>
      )}

      {results.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
          {results.map((item, index) => (
            <article key={index}>
              <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
                <div style={{ flexShrink: 0 }}>
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
                </div>
                
                <div style={{ flexGrow: 1 }}>
                  <header style={{ marginBottom: '10px' }}>
                    <strong>{item.title || 'Untitled'}</strong>
                  </header>
                  <p style={{ marginBottom: '5px' }}><strong>Author:</strong> {item.author || 'Unknown'}</p>
                  <p style={{ marginBottom: '5px' }}><strong>ISBN:</strong> {item.isbn || 'Unknown'}</p>
                  {item.issn && <p style={{ marginBottom: '5px' }}><strong>ISSN:</strong> {item.issn}</p>}
                  {item.subject && <p style={{ marginBottom: '5px' }}><strong>Subject:</strong> {item.subject}</p>}
                  {item.publisher && <p style={{ marginBottom: '5px' }}><strong>Publisher:</strong> {item.publisher}</p>}
                  {item.pub_year && <p style={{ marginBottom: '5px' }}><strong>Year:</strong> {item.pub_year}</p>}

                  {item.holdings && item.holdings.length > 0 && (
                    <div style={{ marginTop: '10px', borderTop: '1px solid #eee', paddingTop: '10px' }}>
                      <small><strong>Holdings:</strong></small>
                      <table style={{ fontSize: '0.85em', marginBottom: 0 }}>
                        <thead>
                          <tr>
                            <th>Location</th>
                            <th>Call Number</th>
                            <th>Status</th>
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
                    Request
                  </button>
                  <button className="secondary outline" onClick={() => showCite(item, 'BibTeX')}>
                    BibTeX
                  </button>
                  <button className="secondary outline" onClick={() => showCite(item, 'RIS')}>
                    RIS
                  </button>
                </div>
              </footer>
            </article>
          ))}
        </div>
      )}

      {/* Citation Modal */}
      {citation && (
        <dialog open>
          <article>
            <header>
              <button aria-label="Close" rel="prev" onClick={() => setCitation(null)}></button>
              <strong>Cite in {citation.format}</strong>
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
                Copy to Clipboard
              </button>
            </footer>
          </article>
        </dialog>
      )}
    </>
  )
}
