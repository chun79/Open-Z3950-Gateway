import React from 'react'
import { Link } from 'react-router-dom'
import { Book, Holding } from '../types'
import { SkeletonCard } from './Skeletons'
import { useI18n } from '../context/I18nContext'

interface SearchResultsProps {
  results: Book[]
  loading: boolean
  targetDB: string
  onRequest: (book: Book) => void
  onCite: (book: Book, format: 'BibTeX' | 'RIS') => void
}

export function SearchResults({ results, loading, targetDB, onRequest, onCite }: SearchResultsProps) {
  const { t } = useI18n()

  const cleanISBN = (isbn: string) => {
    return isbn?.replace(/[^0-9X]/gi, '') || ''
  }

  if (loading) {
    return (
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))', gap: '20px' }}>
        {[1, 2, 3, 4].map(i => <SkeletonCard key={i} />)}
      </div>
    )
  }

  if (results.length === 0) return null

  return (
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
              <p style={{ marginBottom: '5px' }}><strong>{t('search.attr.author')}:</strong> {item.author || 'Unknown'}</p>
              <p style={{ marginBottom: '5px' }}><strong>{t('search.attr.isbn')}:</strong> {item.isbn || 'Unknown'}</p>
              {item.issn && <p style={{ marginBottom: '5px' }}><strong>{t('search.attr.issn')}:</strong> {item.issn}</p>}
              {item.subject && <p style={{ marginBottom: '5px' }}><strong>{t('search.attr.subject')}:</strong> {item.subject}</p>}
              {item.publisher && <p style={{ marginBottom: '5px' }}><strong>{t('detail.publisher')}:</strong> {item.publisher}</p>}
              {item.pub_year && <p style={{ marginBottom: '5px' }}><strong>{t('search.attr.date')}:</strong> {item.pub_year}</p>}

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
                      {item.holdings.map((h: Holding, i: number) => (
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
              <button onClick={() => onRequest(item)}>
                {t('search.action.request')}
              </button>
              <button className="secondary outline" onClick={() => onCite(item, 'BibTeX')}>
                {t('search.action.bibtex')}
              </button>
              <button className="secondary outline" onClick={() => onCite(item, 'RIS')}>
                {t('search.action.ris')}
              </button>
            </div>
          </footer>
        </article>
      ))}
    </div>
  )
}
