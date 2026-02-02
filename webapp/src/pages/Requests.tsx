import { useState, useEffect } from 'react'
import { ILLRequest } from '../types'
import { useAuth } from '../context/AuthContext'

export default function Requests() {
  const [illRequests, setILLRequests] = useState<ILLRequest[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { token, user } = useAuth()

  const fetchILLRequests = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await fetch('/api/ill-requests', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
      if (!response.ok) throw new Error("Failed to fetch requests")
      const data = await response.json()
      setILLRequests(data.data || [])
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleStatusUpdate = async (id: number, status: string) => {
    try {
      const response = await fetch(`/api/ill-requests/${id}/status`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ status })
      })

      if (!response.ok) {
        throw new Error("Failed to update status")
      }
      
      // Refresh list
      fetchILLRequests()
    } catch (err: any) {
      setError(err.message)
    }
  }

  useEffect(() => {
    fetchILLRequests()
  }, [token])

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'approved':
        return <mark style={{ backgroundColor: '#d4edda', color: '#155724', padding: '2px 8px', borderRadius: '4px' }}>✅ Approved</mark>
      case 'rejected':
        return <mark style={{ backgroundColor: '#f8d7da', color: '#721c24', padding: '2px 8px', borderRadius: '4px' }}>❌ Rejected</mark>
      default:
        return <mark style={{ backgroundColor: '#fff3cd', color: '#856404', padding: '2px 8px', borderRadius: '4px' }}>⏳ Pending</mark>
    }
  }

  return (
    <article>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>Inter-Library Loan Requests</strong>
        <button className="outline secondary" onClick={fetchILLRequests} style={{ width: 'auto', marginBottom: 0 }}>
          Refresh
        </button>
      </header>
      
      {error && (
        <article className="pico-background-red-200">
          <strong>❌ Error:</strong> {error}
        </article>
      )}

      {loading ? <p aria-busy="true">Loading...</p> : (
        <figure>
          <table role="grid">
            <thead>
              <tr>
                <th scope="col">ID</th>
                <th scope="col">Library/Target</th>
                <th scope="col">Book Information</th>
                <th scope="col">Status</th>
                {user?.role === 'admin' && <th scope="col">Requestor</th>}
                {user?.role === 'admin' && <th scope="col">Actions</th>}
              </tr>
            </thead>
            <tbody>
              {illRequests.length === 0 ? (
                <tr><td colSpan={user?.role === 'admin' ? 6 : 4} style={{ textAlign: 'center' }}>No requests found.</td></tr>
              ) : (
                illRequests.map((req, idx) => (
                  <tr key={idx}>
                    <td>{req.id}</td>
                    <td><small>{req.target_db}</small></td>
                    <td>
                      <strong>{req.title}</strong><br/>
                      <small>by {req.author} (ISBN: {req.isbn})</small>
                    </td>
                    <td>{getStatusBadge(req.status)}</td>
                    {user?.role === 'admin' && <td>{req.requestor}</td>}
                    {user?.role === 'admin' && (
                      <td>
                        {req.status === 'pending' && (
                          <div role="group">
                            <button className="outline" onClick={() => handleStatusUpdate(req.id, 'approved')} style={{ padding: '5px 10px', fontSize: '0.8em' }}>Approve</button>
                            <button className="outline secondary" onClick={() => handleStatusUpdate(req.id, 'rejected')} style={{ padding: '5px 10px', fontSize: '0.8em' }}>Reject</button>
                          </div>
                        )}
                      </td>
                    )}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </figure>
      )}
    </article>
  )
}
