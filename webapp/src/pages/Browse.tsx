import React, { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { SkeletonCard } from '../components/Skeletons'

interface BookSummary {
  id: string
  title: string
  author: string
  isbn: string
  publisher: string
  pub_year: string
  subjects?: string
}

export default function Browse() {
  const { token } = useAuth()
  const [newArrivals, setNewArrivals] = useState<BookSummary[]>([])
  const [popular, setPopular] = useState<BookSummary[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        // Fetch New Arrivals (simulated API call for now, assuming Search API supports sort)
        // In real implementation, we would call /api/discovery/new
        // Since we don't have dedicated discovery endpoints in main.go yet (Phase 5 plan),
        // we will use the 'scan' API or a search with sort.
        // Wait, we implemented GetNewArrivals in SQLiteProvider but didn't expose it in main.go router yet?
        // Ah, Phase 5 Plan Action 2 said "Upgrade API", but I only modified Provider interface and SQLite implementation.
        // I haven't added the route handlers in main.go for /api/discovery/*.
        // Let's implement client-side simulation for now using Search API with sort if possible,
        // OR, just implement the UI assuming the API exists (and I will add it next).
        
        // Let's stick to the plan: I will add the routes in main.go right after this.
        // So here I will call the new endpoints.

        const resNew = await fetch('/api/discovery/new?limit=10', { headers: { 'Authorization': `Bearer ${token}` } })
        const dataNew = await resNew.json()
        if (dataNew.status === 'success') setNewArrivals(dataNew.data || [])

        const resPop = await fetch('/api/discovery/popular?limit=10', { headers: { 'Authorization': `Bearer ${token}` } })
        const dataPop = await resPop.json()
        if (dataPop.status === 'success') setPopular(dataPop.data || [])

      } catch (err) {
        console.error(err)
      } finally {
        setLoading(false)
      }
    }
    fetchData()
  }, [token])

  const BookRow = ({ title, books }: { title: string, books: BookSummary[] }) => (
    <section style={{ marginBottom: '40px' }}>
      <h3 style={{ borderBottom: '2px solid var(--pico-primary-background)', paddingBottom: '10px', marginBottom: '20px' }}>
        {title}
      </h3>
      
      {books.length === 0 ? (
        <p style={{ color: '#666', fontStyle: 'italic' }}>No books found.</p>
      ) : (
        <div style={{ 
          display: 'flex', 
          overflowX: 'auto', 
          gap: '20px', 
          paddingBottom: '20px',
          scrollSnapType: 'x mandatory' 
        }}>
          {books.map(book => (
            <Link 
              key={book.id} 
              to={`/book/Default/${encodeURIComponent(book.id)}`} 
              style={{ 
                textDecoration: 'none', 
                color: 'inherit',
                flex: '0 0 160px',
                scrollSnapAlign: 'start'
              }}
            >
              <div style={{ 
                height: '240px', 
                background: '#f0f0f0', 
                borderRadius: '8px', 
                marginBottom: '10px',
                backgroundImage: `url(https://covers.openlibrary.org/b/isbn/${book.isbn?.replace(/[^0-9X]/gi, '')}-M.jpg?default=https://placehold.co/160x240/e0e0e0/808080?text=No+Cover)`,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
                boxShadow: '0 4px 6px rgba(0,0,0,0.1)'
              }} />
              <strong style={{ display: 'block', fontSize: '0.9em', lineHeight: '1.2em', height: '2.4em', overflow: 'hidden' }}>{book.title}</strong>
              <small style={{ color: '#666' }}>{book.author}</small>
            </Link>
          ))}
        </div>
      )}
    </section>
  )

  if (loading) return <div className="container"><SkeletonCard /><SkeletonCard /></div>

  return (
    <div className="container">
      <header style={{ textAlign: 'center', marginBottom: '40px', padding: '40px 0', background: 'var(--pico-card-background-color)', borderRadius: '12px' }}>
        <h1>🏛 Library Discovery</h1>
        <p>Explore our curated collections and new arrivals.</p>
      </header>

      <BookRow title="✨ New Arrivals" books={newArrivals} />
      <BookRow title="🔥 Popular & Trending" books={popular} />
      
      <section>
        <h3>📚 Browse by Subject</h3>
        <div className="grid">
          {['Computer Science', 'History', 'Science', 'Fiction', 'Art', 'Business'].map(sub => (
            <Link 
              key={sub}
              to={`/?query=${sub}&attr1=21`} // Subject search
              role="button" 
              className="outline contrast"
              style={{ padding: '20px', fontSize: '1.2em' }}
            >
              {sub}
            </Link>
          ))}
        </div>
      </section>
    </div>
  )
}
