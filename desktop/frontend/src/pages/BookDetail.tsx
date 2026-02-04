import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useI18n } from '../context/I18nContext'
import { Book } from '../types'
import { GetBookDetails, RequestILL } from '../../wailsjs/go/main/App'

export default function BookDetail() {
  const { db, id } = useParams<{ db: string, id: string }>()
  const navigate = useNavigate()
  const { t } = useI18n()
  
  const [book, setBook] = useState<Book | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [comments, setComments] = useState('')
  const [requestStatus, setRequestStatus] = useState<{msg: string, type: 'success' | 'error'} | null>(null)
  
  // New State for Modals
  const [showMarc, setShowMarc] = useState(false)
  const [citation, setCitation] = useState<{format: string, text: string} | null>(null)

  useEffect(() => {
    if (!db || !id) return
    
    const fetchBook = async () => {
// ... existing fetch logic ...
    }
    fetchBook()
  }, [db, id])

  const generateCitation = (format: string) => {
    if (!book) return
    const date = new Date().getFullYear()
    let text = ""
    if (format === "APA") {
      text = `${book.author}. (${book.pub_year || 'n.d.'}). ${book.title}. ${book.publisher}.`
    } else if (format === "MLA") {
      text = `${book.author}. ${book.title}. ${book.publisher}, ${book.pub_year || 'n.d.'}.`
    } else if (format === "BibTeX") {
      text = `@book{${cleanISBN(book.isbn) || 'book'},\n  title = {${book.title}},\n  author = {${book.author}},\n  year = {${book.pub_year || '2026'}},\n  publisher = {${book.publisher}}\n}`
    }
    setCitation({ format, text })
  }

  const handleILLRequest = async () => {
// ... existing ILL logic ...
  }

  const cleanISBN = (isbn: string) => {
    return isbn?.replace(/[^0-9X]/gi, '') || ''
  }

  if (loading) return <article aria-busy="true"></article>
  if (error) return <article className="pico-background-red-200">Error: {error}</article>
  if (!book) return <article>Book not found</article>

  return (
    <div className="container">
      <button className="outline secondary" onClick={() => navigate(-1)} style={{marginBottom: '20px'}}>
        ← {t('detail.back')}
      </button>
      
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
                <small>{t('detail.publisher')}</small>
                <p><strong>{book.publisher}</strong></p>
              </div>
              <div>
                <small>{t('detail.edition')}</small>
                <p><strong>{book.edition || '-'}</strong></p>
              </div>
              <div>
                <small>{t('detail.isbn')}</small>
                <p><strong>{book.isbn}</strong></p>
              </div>
            </div>

            {book.summary && (
              <details open>
                <summary>{t('detail.summary')}</summary>
                <p>{book.summary}</p>
              </details>
            )}

            {book.toc && (
              <details>
                <summary>{t('detail.toc')}</summary>
                <pre style={{whiteSpace: 'pre-wrap', fontFamily: 'sans-serif'}}>{book.toc}</pre>
              </details>
            )}

            {book.physical && (
              <p><small>{t('detail.physical')}: {book.physical}</small></p>
            )}
            
            {book.series && (
              <p><small>{t('detail.series')}: {book.series}</small></p>
            )}

            <hr />
            
            <label htmlFor="comments">
              {t('detail.comments')}
              <textarea 
                id="comments" 
                value={comments} 
                onChange={(e) => setComments(e.target.value)} 
                placeholder="e.g. Need by Friday, Chapter 3 only..."
                style={{ resize: 'vertical', minHeight: '80px' }}
              />
            </label>

            <footer>
              <div role="group">
                <button onClick={handleILLRequest}>{t('detail.request_btn')}</button>
                <button className="secondary outline">Add to List</button>
                <button className="secondary outline" onClick={() => generateCitation('APA')}>❞ Cite</button>
                {book.fields && <button className="contrast outline" onClick={() => setShowMarc(true)}>📜 MARC</button>}
              </div>
            </footer>
          </div>
        </div>
      </article>

      {/* MARC Modal */}
      {showMarc && (
        <dialog open>
          <article style={{ width: '100%', maxWidth: '800px' }}>
            <header>
              <button aria-label="Close" rel="prev" onClick={() => setShowMarc(false)}></button>
              <strong>MARC Record Inspection</strong>
            </header>
            <div style={{ maxHeight: '60vh', overflowY: 'auto' }}>
              <table role="grid">
                <thead>
                  <tr>
                    <th>Tag</th>
                    <th>Value</th>
                  </tr>
                </thead>
                <tbody>
                  {book.fields?.map((f: any, idx: number) => (
                    <tr key={idx}>
                      <td><kbd>{f.Tag}</kbd></td>
                      <td style={{ fontFamily: 'monospace', fontSize: '0.9em' }}>{f.Value}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </article>
        </dialog>
      )}

      {/* Citation Modal */}
      {citation && (
        <dialog open>
          <article>
            <header>
              <button aria-label="Close" rel="prev" onClick={() => setCitation(null)}></button>
              <strong>Cite in {citation.format}</strong>
            </header>
            <textarea readOnly value={citation.text} style={{ height: '150px' }} />
            <footer>
              <div role="group">
                <button onClick={() => { navigator.clipboard.writeText(citation.text); setCitation(null) }}>Copy</button>
                <button className="secondary outline" onClick={() => generateCitation(citation.format === 'APA' ? 'MLA' : citation.format === 'MLA' ? 'BibTeX' : 'APA')}>
                  Switch Format
                </button>
              </div>
            </footer>
          </article>
        </dialog>
      )}
    </div>
  )
}
