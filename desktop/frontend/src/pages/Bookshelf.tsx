import React, { useState, useEffect } from 'react'
import { Trash2, Book as BookIcon, Download } from 'lucide-react'
import toast from 'react-hot-toast'
import { useI18n } from '../context/I18nContext'
import { ListSavedBooks, DeleteSavedBook, ExportBookshelf } from '../../wailsjs/go/main/App'

interface SavedBook {
  id: number
  title: string
  author: string
  isbn: string
  source_db: string
  saved_at: string
}

export default function Bookshelf() {
  const { t } = useI18n()
  const [books, setBooks] = useState<SavedBook[]>([])
  const [filter, setFilter] = useState('')
  
  const fetchBooks = () => {
    ListSavedBooks().then((data: any) => setBooks(data || []))
  }

  const handleDelete = (id: number) => {
// ...
  }

  const handleExport = async () => {
// ...
  }

  useEffect(() => {
    fetchBooks()
  }, [])

  const filteredBooks = books.filter(b => 
    b.title.toLowerCase().includes(filter.toLowerCase()) || 
    b.author.toLowerCase().includes(filter.toLowerCase()) ||
    b.source_db.toLowerCase().includes(filter.toLowerCase())
  )

  return (
    <article>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>📚 My Local Bookshelf</strong>
        {books.length > 0 && (
          <div style={{ display: 'flex', gap: '10px' }}>
            <input 
              type="search" 
              placeholder="Filter..." 
              value={filter}
              onChange={e => setFilter(e.target.value)}
              style={{ marginBottom: 0, padding: '4px 8px', fontSize: '0.9em', width: '200px' }}
            />
            <button 
              className="outline secondary" 
              onClick={handleExport}
              style={{ padding: '4px 12px', fontSize: '0.8em', marginBottom: 0, display: 'flex', alignItems: 'center', gap: '5px' }}
            >
              <Download size={16} /> Export CSV
            </button>
          </div>
        )}
      </header>
      
      {books.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '40px 20px', color: 'var(--pico-muted-color)' }}>
          <BookIcon size={48} style={{ opacity: 0.5, marginBottom: '10px' }} />
          <p>No books saved yet.</p>
          <small>Go to Search and save some books to build your collection!</small>
        </div>
      ) : filteredBooks.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '20px', color: 'var(--pico-muted-color)' }}>
          <p>No matches for "{filter}"</p>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '20px' }}>
          {filteredBooks.map(b => (
            <article key={b.id} style={{ marginBottom: 0, position: 'relative' }}>
              <header style={{ padding: '10px', fontSize: '0.8em', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ opacity: 0.7 }}>{b.source_db}</span>
                <button 
                  className="outline secondary"
                  onClick={() => handleDelete(b.id)} 
                  style={{ border: 'none', padding: '4px', display: 'flex', alignItems: 'center' }}
                  title="Remove"
                >
                  <Trash2 size={16} />
                </button>
              </header>
              <div style={{ padding: '15px' }}>
                <h5>{b.title}</h5>
                <small>{b.author}</small>
                <br/>
                <small style={{ color: 'var(--pico-muted-color)' }}>ISBN: {b.isbn}</small>
              </div>
            </article>
          ))}
        </div>
      )}
    </article>
  )
}
