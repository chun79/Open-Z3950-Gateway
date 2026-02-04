import React, { useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { SkeletonCard } from '../components/Skeletons'

interface PatronProfile {
  AE: string // Personal Name
  BZ: string // Hold Items Limit
  // Add other SIP2 fields as needed
}

export default function MyLibrary() {
  const { token } = useAuth()
  const [barcode, setBarcode] = useState('')
  const [password, setPassword] = useState('')
  const [isLoggedIn, setIsLoggedIn] = useState(false)
  const [profile, setProfile] = useState<PatronProfile | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const res = await fetch('/api/ils/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ barcode, password })
      })

      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error || 'Login failed')
      }

      setIsLoggedIn(true)
      fetchProfile()
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const fetchProfile = async () => {
    try {
      const res = await fetch(`/api/ils/profile?barcode=${encodeURIComponent(barcode)}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      const data = await res.json()
      if (data.status === 'success') {
        setProfile(data.data)
      }
    } catch (err) {
      console.error(err)
    }
  }

  const handleLogout = () => {
    setIsLoggedIn(false)
    setProfile(null)
    setBarcode('')
    setPassword('')
  }

  if (!isLoggedIn) {
    return (
      <article>
        <header>
          <strong>🏛 Login to Library System</strong>
        </header>
        <form onSubmit={handleLogin}>
          <label>
            Library Card Barcode
            <input 
              type="text" 
              value={barcode}
              onChange={e => setBarcode(e.target.value)}
              placeholder="Enter your card number"
              required
            />
          </label>
          <label>
            Password / PIN
            <input 
              type="password" 
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="Enter your PIN"
              required
            />
          </label>
          {error && <small style={{ color: 'red', display: 'block', marginBottom: '10px' }}>{error}</small>}
          <button type="submit" disabled={loading}>
            {loading ? 'Verifying...' : 'Link Card'}
          </button>
        </form>
        <p style={{fontSize: '0.8em', color: '#666', marginTop: '10px'}}>
          Connect your physical library card to view loans, renewals, and fines.
        </p>
      </article>
    )
  }

  return (
    <article>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>My Library Card</strong>
        <button className="outline secondary" onClick={handleLogout} style={{fontSize: '0.8em', padding: '5px 10px'}}>Unlink</button>
      </header>

      {profile ? (
        <div className="grid">
          <div>
            <h5>Card Holder</h5>
            <p style={{ fontSize: '1.2em' }}>{profile.AE || 'Unknown'}</p>
            <p style={{ color: '#666' }}>Card: {barcode}</p>
          </div>
          <div>
            <h5>Status</h5>
            <ul>
              <li><strong>Hold Limit:</strong> {profile.BZ || 'N/A'}</li>
              <li><strong>Valid:</strong> ✅ Yes</li>
            </ul>
          </div>
        </div>
      ) : (
        <SkeletonCard />
      )}

      <footer style={{marginTop: '20px'}}>
        <div className="grid">
          <button className="secondary outline">View Loans (0)</button>
          <button className="secondary outline">Pay Fines ($0.00)</button>
        </div>
      </footer>
    </article>
  )
}
