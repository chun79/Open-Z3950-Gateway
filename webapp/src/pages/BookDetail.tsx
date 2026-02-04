import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { Book } from '../types'

export default function BookDetail() {
  const { db, id } = useParams<{ db: string, id: string }>()
  const navigate = useNavigate()
  const { token } = useAuth()
  const { t } = useI18n()
  
  const [book, setBook] = useState<Book | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [comments, setComments] = useState('')
  const [requestStatus, setRequestStatus] = useState<{msg: string, type: 'success' | 'error'} | null>(null)

  const isLocal = db === 'Default'

  useEffect(() => {
    if (!db || !id) return
    const fetchBook = async () => {
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
    fetchBook()
  }, [db, id, token])

  const handleILLRequest = async () => {
    if (!book) return
    try {
      const response = await fetch('/api/ill-requests', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ title: book.title, author: book.author, isbn: book.isbn, target_db: db, record_id: book.record_id || id, comments })
      })
      if (!response.ok) throw new Error('Request failed')
      setRequestStatus({ msg: t('detail.request_success').replace('{title}', book.title), type: 'success' })
    } catch (err: any) { setRequestStatus({ msg: t('detail.request_fail').replace('{error}', err.message), type: 'error' }) }
  }

  if (loading) return <article aria-busy="true"></article>
  if (error) return <article className="pico-background-red-200">Error: {error}</article>
  if (!book) return <article>Book not found</article>

  return (
    <div className="container">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <button className="outline secondary" onClick={() => navigate(-1)} style={{ marginBottom: 0 }}>← {t('detail.back')}</button>
        {isLocal && (
          <Link to={`/edit/${db}/${encodeURIComponent(id || '')}`} role="button" className="contrast">
            ✍️ Edit Metadata
          </Link>
        )}
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
            <div className="grid">
              <div><small>{t('detail.publisher')}</small><p><strong>{book.publisher}</strong></p></div>
              <div><small>{t('detail.isbn')}</small><p><strong>{book.isbn}</strong></p></div>
              <div><small>Source</small><p><mark>{db}</mark></p></div>
            </div>
            {book.summary && <details open><summary>{t('detail.summary')}</summary><p>{book.summary}</p></details>}
            <hr />
            {!isLocal && (
              <>
                <label htmlFor="comments">{t('detail.comments')}
                  <textarea id="comments" value={comments} onChange={(e) => setComments(e.target.value)} placeholder="e.g. Need by Friday..." style={{ resize: 'vertical', minHeight: '80px' }} />
                </label>
                <button onClick={handleILLRequest}>{t('detail.request_btn')}</button>
              </>
            )}
            {isLocal && <p>✅ This record is in your local collection.</p>}
          </div>
        </div>
      </article>
    </div>
  )
}