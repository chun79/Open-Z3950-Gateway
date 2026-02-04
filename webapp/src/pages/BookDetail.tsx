import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { Book, Holding } from '../types'
import toast from 'react-hot-toast'

export default function BookDetail() {
  const { db, id } = useParams<{ db: string, id: string }>()
  const navigate = useNavigate()
  const { token } = useAuth()
  const { t } = useI18n()
  
  const [book, setBook] = useState<Book | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  
  // AI State
  const [insight, setInsight] = useState<string | null>(null)
  const [generatingAI, setGeneratingAI] = useState(false)

  // Item Management State
  const [showAddItem, setShowAddItem] = useState(false)
  const [newBarcode, setNewBarcode] = useState('')
  const [newLocation, setNewLocation] = useState('Main Stack')

  const isLocal = db === 'Default'

  const fetchBook = async () => {
    if (!db || !id) return
    setLoading(true)
    try {
      const response = await fetch(`/api/books/${db}/${encodeURIComponent(id)}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!response.ok) throw new Error("Failed to load book details")
      const data = await response.json()
      setBook(data.data)
    } catch (err: any) { setError(err.message) } finally { setLoading(false) }
  }

  useEffect(() => {
    fetchBook()
  }, [db, id, token])

  const generateAIInsight = async () => {
    setGeneratingAI(true)
    try {
      const res = await fetch(`/api/books/${db}/${encodeURIComponent(id || '')}/ai-insight`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      const data = await res.json()
      if (res.ok) {
        setInsight(data.insight)
      } else {
        throw new Error(data.error || "AI Generation failed")
      }
    } catch (err: any) {
      toast.error(err.message)
    } finally {
      setGeneratingAI(false)
    }
  }

  const handleAddItem = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await fetch('/api/items', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ bib_id: id, barcode: newBarcode, location: newLocation, call_number: book?.record_id })
      })
      if (!res.ok) throw new Error("Failed to add item")
      toast.success("Item added!")
      setNewBarcode('')
      setShowAddItem(false)
      fetchBook()
    } catch (err: any) { toast.error(err.message) }
  }

  if (loading) return <article aria-busy="true"></article>
  if (error) return <article className="pico-background-red-200">Error: {error}</article>
  if (!book) return <article>Book not found</article>

  return (
    <div className="container">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <button className="outline secondary" onClick={() => navigate(-1)} style={{ marginBottom: 0 }}>← {t('detail.back')}</button>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button className="contrast outline" onClick={generateAIInsight} disabled={generatingAI} style={{ marginBottom: 0 }}>
            {generatingAI ? '✨ Analyzing...' : '✨ AI Insight'}
          </button>
          {isLocal && (
            <Link to={`/edit/${db}/${encodeURIComponent(id || '')}`} role="button" className="contrast">
              ✍️ Edit
            </Link>
          )}
        </div>
      </div>
      
      <article>
        <div className="grid">
          <div style={{ flex: '0 0 200px' }}>
             <img src={book.isbn ? `https://covers.openlibrary.org/b/isbn/${book.isbn.replace(/[^0-9X]/gi, '')}-L.jpg?default=https://placehold.co/200x300/e0e0e0/808080?text=No+Cover` : 'https://placehold.co/200x300/e0e0e0/808080?text=No+ISBN'} style={{ width: '100%', borderRadius: '8px' }} />
          </div>
          <div>
            <hgroup>
              <h2>{book.title}</h2>
              <h3>{book.author}</h3>
            </hgroup>

            {/* AI Insight Section */}
            {insight && (
              <div style={{ padding: '20px', background: 'linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%)', borderRadius: '12px', marginBottom: '20px', borderLeft: '5px solid #1976d2' }}>
                <strong style={{ color: '#1976d2' }}>✨ AI Librarian Insight</strong>
                <pre style={{ whiteSpace: 'pre-wrap', fontFamily: 'inherit', margin: '10px 0 0 0', fontSize: '0.95em', color: '#333' }}>
                  {insight}
                </pre>
              </div>
            )}

            <div className="grid">
              <div><small>Publisher</small><p><strong>{book.publisher}</strong></p></div>
              <div><small>ISBN</small><p><strong>{book.isbn}</strong></p></div>
              <div><small>Source</small><p><mark>{db}</mark></p></div>
            </div>
            
            <hr />

            <section>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h4>🏛 Holdings</h4>
                {isLocal && <button className="outline secondary" onClick={() => setShowAddItem(!showAddItem)} style={{ padding: '2px 10px', fontSize: '0.8em' }}>{showAddItem ? 'Cancel' : '+ Add'}</button>}
              </div>

              {showAddItem && (
                <form onSubmit={handleAddItem} style={{ padding: '15px', background: '#f0f2f5', borderRadius: '8px', marginBottom: '15px' }}>
                  <div className="grid">
                    <input type="text" placeholder="Barcode" value={newBarcode} onChange={e => setNewBarcode(e.target.value)} required />
                    <button type="submit">Confirm</button>
                  </div>
                </form>
              )}

              {book.holdings && book.holdings.length > 0 ? (
                <table className="striped">
                  <tbody>
                    {book.holdings.map((h, i) => (
                      <tr key={i}>
                        <td><code>{h.call_number}</code></td>
                        <td>{h.location}</td>
                        <td><ins style={{ color: h.status === 'Available' ? 'green' : 'orange', textDecoration: 'none' }}>{h.status}</ins></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : <p style={{ color: '#666', fontStyle: 'italic' }}>No physical items.</p>}
            </section>
          </div>
        </div>
      </article>
    </div>
  )
}