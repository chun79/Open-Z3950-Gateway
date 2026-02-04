import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import toast from 'react-hot-toast'

interface ILLRequest {
  id: number
  title: string
  author: string
  isbn: string
  target_db: string
  status: string
  requestor: string
}

interface Stats {
  total_titles: number
  total_items: number
  active_loans: number
  overdue_loans: number
}

export default function AdminDashboard() {
  const { token } = useAuth()
  const [requests, setRequests] = useState<ILLRequest[]>([])
  const [stats, setStats] = useState<Stats | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'requests' | 'stats'>('stats')

  const fetchData = async () => {
    setLoading(true)
    try {
      // 1. Fetch ILL Requests
      const resReq = await fetch('/api/ill-requests', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      const dataReq = await resReq.json()
      setRequests(dataReq.data || [])

      // 2. Fetch Stats
      const resStats = await fetch('/api/admin/stats', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      const dataStats = await resStats.json()
      if (dataStats.status === 'success') {
        setStats(dataStats.data)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [token])

  const updateStatus = async (id: number, status: string) => {
    try {
      const res = await fetch(`/api/ill-requests/${id}/status`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ status })
      })
      if (res.ok) {
        toast.success(`Request marked as ${status}`)
        fetchData()
      }
    } catch (err) {
      toast.error("Failed to update status")
    }
  }

  if (loading) return <article aria-busy="true"></article>

  return (
    <div className="container">
      <header style={{ marginBottom: '30px' }}>
        <h2>🛡 LSP Admin Command Center</h2>
        <nav>
          <ul>
            <li>
              <button 
                className={activeTab === 'stats' ? '' : 'outline secondary'} 
                onClick={() => setActiveTab('stats')}
                style={{ padding: '5px 20px' }}
              >
                Collection Insights
              </button>
            </li>
            <li>
              <button 
                className={activeTab === 'requests' ? '' : 'outline secondary'} 
                onClick={() => setActiveTab('requests')}
                style={{ padding: '5px 20px' }}
              >
                ILL Requests ({requests.filter(r => r.status === 'pending').length})
              </button>
            </li>
          </ul>
        </nav>
      </header>

      {activeTab === 'stats' && stats && (
        <section>
          <div className="grid">
            <article style={{ padding: '20px', textAlign: 'center', background: '#f8f9fa' }}>
              <small style={{ color: '#666', textTransform: 'uppercase' }}>Total Bibliographies</small>
              <h1 style={{ margin: '10px 0', fontSize: '3em' }}>{stats.total_titles}</h1>
              <p style={{ fontSize: '0.8em', opacity: 0.7 }}>Unique titles in catalog</p>
            </article>
            <article style={{ padding: '20px', textAlign: 'center', background: '#f8f9fa' }}>
              <small style={{ color: '#666', textTransform: 'uppercase' }}>Total Physical Items</small>
              <h1 style={{ margin: '10px 0', fontSize: '3em' }}>{stats.total_items}</h1>
              <p style={{ fontSize: '0.8em', opacity: 0.7 }}>Total barcoded items</p>
            </article>
            <article style={{ padding: '20px', textAlign: 'center', background: '#e3f2fd', border: '1px solid #90caf9' }}>
              <small style={{ color: '#1976d2', textTransform: 'uppercase' }}>Active Loans</small>
              <h1 style={{ margin: '10px 0', fontSize: '3em', color: '#1976d2' }}>{stats.active_loans}</h1>
              <p style={{ fontSize: '0.8em', color: '#1976d2' }}>Books currently out</p>
            </article>
            <article style={{ padding: '20px', textAlign: 'center', background: '#fff1f0', border: '1px solid #ffa39e' }}>
              <small style={{ color: '#f5222d', textTransform: 'uppercase' }}>Overdue Items</small>
              <h1 style={{ margin: '10px 0', fontSize: '3em', color: '#f5222d' }}>{stats.overdue_loans}</h1>
              <p style={{ fontSize: '0.8em', color: '#f5222d' }}>Requires intervention</p>
            </article>
          </div>

          <article style={{ marginTop: '20px' }}>
            <header>📈 Growth & Circulation Trends</header>
            <p style={{ color: '#666', fontStyle: 'italic' }}>Historical trend data will be available as circulation volume increases.</p>
            <div style={{ height: '10px', width: `${(stats.active_loans / stats.total_items) * 100}%`, background: '#1976d2', borderRadius: '5px' }}></div>
            <small>Collection Utilization: {((stats.active_loans / stats.total_items) * 100).toFixed(1)}%</small>
          </article>
        </section>
      )}

      {activeTab === 'requests' && (
        <article>
          <header>Pending Inter-Library Loan Requests</header>
          {requests.length === 0 ? (
            <p>No active requests.</p>
          ) : (
            <table className="striped">
              <thead>
                <tr>
                  <th>Title / ISBN</th>
                  <th>Requestor</th>
                  <th>Source DB</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {requests.map(r => (
                  <tr key={r.id}>
                    <td>
                      <strong>{r.title}</strong><br/>
                      <small>{r.isbn}</small>
                    </td>
                    <td>{r.requestor}</td>
                    <td><mark>{r.target_db}</mark></td>
                    <td>
                      <span className={`badge ${r.status}`}>{r.status}</span>
                    </td>
                    <td>
                      {r.status === 'pending' && (
                        <div role="group">
                          <button className="outline" onClick={() => updateStatus(r.id, 'approved')} style={{ padding: '2px 10px', fontSize: '0.8em' }}>Approve</button>
                          <button className="outline secondary" onClick={() => updateStatus(r.id, 'rejected')} style={{ padding: '2px 10px', fontSize: '0.8em' }}>Reject</button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </article>
      )}
    </div>
  )
}