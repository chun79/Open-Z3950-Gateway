import React, { useState, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { SkeletonCard } from '../components/Skeletons'

type ILLRequest = {
  id: number
  target_db: string
  record_id: string
  title: string
  author: string
  isbn: string
  status: 'pending' | 'approved' | 'rejected'
  requestor: string
  comments?: string
  created_at?: string
}

export default function AdminDashboard() {
  const { token, user } = useAuth()
  const { t } = useI18n()
  const [requests, setRequests] = useState<ILLRequest[]>([])
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState<string>('all')

  const fetchRequests = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/ill-requests', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      const data = await res.json()
      if (data.status === 'success') {
        setRequests(data.data || [])
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (token) fetchRequests()
  }, [token])

  const handleStatusUpdate = async (id: number, newStatus: string) => {
    if (!confirm(`Are you sure you want to mark this request as ${newStatus}?`)) return

    try {
      const res = await fetch(`/api/ill-requests/${id}/status`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ status: newStatus })
      })
      
      if (res.ok) {
        // Optimistic update
        setRequests(requests.map(r => r.id === id ? { ...r, status: newStatus as any } : r))
      } else {
        alert("Failed to update status")
      }
    } catch (err) {
      console.error(err)
      alert("Network error")
    }
  }

  const filteredRequests = requests.filter(r => {
    if (filter === 'all') return true
    return r.status === filter
  })

  if (!user || user.role !== 'admin') {
    return (
      <article className="pico-background-red-200">
        <h3>⛔ Access Denied</h3>
        <p>You need administrator privileges to view this page.</p>
      </article>
    )
  }

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <hgroup style={{ marginBottom: 0 }}>
          <h2>Admin Dashboard</h2>
          <p>Manage Inter-Library Loan Requests</p>
        </hgroup>
        
        <div role="group">
          <button className={filter === 'all' ? '' : 'outline'} onClick={() => setFilter('all')}>All</button>
          <button className={filter === 'pending' ? '' : 'outline'} onClick={() => setFilter('pending')}>Pending</button>
          <button className={filter === 'approved' ? '' : 'outline'} onClick={() => setFilter('approved')}>Approved</button>
        </div>
      </div>

      {loading ? (
        <SkeletonCard />
      ) : (
        <article style={{ padding: 0, overflow: 'hidden' }}>
          <table className="striped" style={{ marginBottom: 0 }}>
            <thead style={{ backgroundColor: '#f9f9f9' }}>
              <tr>
                <th style={{ padding: '15px' }}>ID</th>
                <th>Requestor</th>
                <th>Book Details</th>
                <th>Status</th>
                <th style={{ textAlign: 'right', paddingRight: '20px' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredRequests.length === 0 ? (
                <tr>
                  <td colSpan={5} style={{ textAlign: 'center', padding: '30px', color: '#888' }}>
                    No requests found.
                  </td>
                </tr>
              ) : (
                filteredRequests.map(req => (
                  <tr key={req.id}>
                    <td style={{ padding: '15px' }}>#{req.id}</td>
                    <td>
                      <strong>{req.requestor}</strong>
                      {req.created_at && <div style={{ fontSize: '0.7em', color: '#666' }}>{new Date(req.created_at).toLocaleDateString()}</div>}
                    </td>
                    <td>
                      <div><strong>{req.title}</strong></div>
                      <div style={{ fontSize: '0.85em' }}>{req.author}</div>
                      <div style={{ fontSize: '0.8em', color: '#666' }}>ISBN: {req.isbn}</div>
                      <div style={{ fontSize: '0.8em', color: '#666' }}>Source: {req.target_db}</div>
                    </td>
                    <td>
                      <span 
                        data-tooltip={req.comments || "No comments"}
                        style={{
                          padding: '4px 8px',
                          borderRadius: '4px',
                          fontSize: '0.85em',
                          fontWeight: 'bold',
                          backgroundColor: req.status === 'approved' ? '#d4edda' : req.status === 'rejected' ? '#f8d7da' : '#fff3cd',
                          color: req.status === 'approved' ? '#155724' : req.status === 'rejected' ? '#721c24' : '#856404'
                        }}
                      >
                        {req.status.toUpperCase()}
                      </span>
                    </td>
                    <td style={{ textAlign: 'right', paddingRight: '20px' }}>
                      {req.status === 'pending' && (
                        <div role="group" style={{ display: 'inline-flex', marginBottom: 0 }}>
                          <button 
                            onClick={() => handleStatusUpdate(req.id, 'approved')}
                            style={{ padding: '5px 10px', fontSize: '0.8em', backgroundColor: '#28a745', borderColor: '#28a745' }}
                          >
                            ✓ Approve
                          </button>
                          <button 
                            onClick={() => handleStatusUpdate(req.id, 'rejected')}
                            style={{ padding: '5px 10px', fontSize: '0.8em', backgroundColor: '#dc3545', borderColor: '#dc3545' }}
                          >
                            ✕ Reject
                          </button>
                        </div>
                      )}
                      {req.status !== 'pending' && (
                        <button 
                          className="outline secondary"
                          onClick={() => handleStatusUpdate(req.id, 'pending')}
                          style={{ padding: '5px 10px', fontSize: '0.8em', border: 'none' }}
                        >
                          Reset
                        </button>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </article>
      )}
    </>
  )
}
