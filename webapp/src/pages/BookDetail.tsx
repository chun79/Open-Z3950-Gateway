import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { Book } from '../types'

export default function BookDetail() {
  const { db, id } = useParams<{ db: string, id: string }>()
  const navigate = useNavigate()
  const { token } = useAuth()
  
  const [book, setBook] = useState<Book | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [requestStatus, setRequestStatus] = useState<{msg: string, type: 'success' | 'error'} | null>(null)

  useEffect(() => {
    if (!db || !id) return
    
    // Check if we passed state via navigation (not typical for direct link, but good for back)
    // Actually, always fetch fresh data to ensure we have details
    const fetchBook = async () => {
      setLoading(true)
      try {
        // We need to encode the ID because it might contain colons (sessionID:index)
        const safeId = encodeURIComponent(id)
        const response = await fetch(`/api/books/${db}/${safeId}`, {
          headers: { 'Authorization': `Bearer ${token}` }
        })
        if (!response.ok) throw new Error("Failed to load book details")
        const data = await response.json()
        setBook(data.data)
      } catch (err: any) {
        setError(err.message)
      } finally {
        setLoading(false)
      }
    }
    fetchBook()
  }, [db, id, token])

  const handleILLRequest = async () => {
    if (!book) return
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
          target_db: db || 'unknown',
          record_id: book.record_id || id || 'unknown'
        })
      })

      if (!response.ok) {
        const errData = await response.json()
        throw new Error(errData.error || 'Request failed')
      }

      setRequestStatus({ msg: `Requested "${book.title}" successfully!`, type: 'success' })
    } catch (err: any) {
      setRequestStatus({ msg: `Failed to request: ${err.message}`, type: 'error' })
    }
  }

  const cleanISBN = (isbn: string) => {
    return isbn?.replace(/[^0-9X]/gi, '') || ''
  }

  if (loading) return <article aria-busy="true"></article>
  if (error) return <article className="pico-background-red-200">Error: {error}</article>
  if (!book) return <article>Book not found</article>

  return (
    <div className="container">
      <button className="outline secondary" onClick={() => navigate(-1)} style={{marginBottom: '20px'}}>← Back</button>
      
      <article>
        <div className="grid">
          <div style={{ flex: '0 0 200px' }}>
             <img 
                src={book.isbn 
                  ? `https://covers.openlibrary.org/b/isbn/${cleanISBN(book.isbn)}-L.jpg?default=https://placehold.co/200x300/e0e0e0/808080?text=No+Cover`
                  : 'https://placehold.co/200x300/e0e0e0/808080?text=No+ISBN'
                }
                alt="Cover"
                style={{ width: '100%', borderRadius: '8px', boxShadow: '0 4px 6px rgba(0,0,0,0.1)' }}
              />
          </div>
          <div>
            <hgroup>
              <h2>{book.title}</h2>
              <h3>{book.author}</h3>
            </hgroup>
            
            {requestStatus && (
              <div style={{ 
                padding: '10px', 
                marginBottom: '15px',
                borderRadius: '4px',
                backgroundColor: requestStatus.type === 'success' ? '#d4edda' : '#f8d7da',
                color: requestStatus.type === 'success' ? '#155724' : '#721c24'
              }}>
                {requestStatus.msg}
              </div>
            )}

            <div className="grid">
              <div>
                <small>Publisher</small>
                <p><strong>{book.publisher}</strong></p>
              </div>
              <div>
                <small>Edition</small>
                <p><strong>{book.edition || '-'}</strong></p>
              </div>
              <div>
                <small>ISBN</small>
                <p><strong>{book.isbn}</strong></p>
              </div>
            </div>

            {book.summary && (
              <details open>
                <summary>Summary</summary>
                <p>{book.summary}</p>
              </details>
            )}

            {book.toc && (
              <details>
                <summary>Table of Contents</summary>
                <pre style={{whiteSpace: 'pre-wrap', fontFamily: 'sans-serif'}}>{book.toc}</pre>
              </details>
            )}

            {book.physical && (
              <p><small>Physical: {book.physical}</small></p>
            )}
            
            {book.series && (
              <p><small>Series: {book.series}</small></p>
            )}

            <footer>
              <div role="group">
                <button onClick={handleILLRequest}>Request Item</button>
                <button className="secondary outline">Add to List</button>
              </div>
            </footer>
          </div>
        </div>
      </article>
    </div>
  )
}
