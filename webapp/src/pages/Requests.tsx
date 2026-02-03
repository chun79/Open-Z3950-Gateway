import { useState, useEffect } from 'react'
import { ILLRequest } from '../types'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { SkeletonRow } from '../components/Skeletons'

export default function Requests() {
  const [illRequests, setILLRequests] = useState<ILLRequest[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { token, user } = useAuth()
  const { t } = useI18n()

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
        return <mark style={{ backgroundColor: '#d4edda', color: '#155724', padding: '2px 8px', borderRadius: '4px' }}>✅ {t('requests.status.approved')}</mark>
      case 'rejected':
        return <mark style={{ backgroundColor: '#f8d7da', color: '#721c24', padding: '2px 8px', borderRadius: '4px' }}>❌ {t('requests.status.rejected')}</mark>
      default:
        return <mark style={{ backgroundColor: '#fff3cd', color: '#856404', padding: '2px 8px', borderRadius: '4px' }}>⏳ {t('requests.status.pending')}</mark>
    }
  }

  return (
    <article>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>{t('requests.title')}</strong>
        <button className="outline secondary" onClick={fetchILLRequests} style={{ width: 'auto', marginBottom: 0 }}>
          {t('requests.refresh')}
        </button>
      </header>
      
      {error && (
        <article className="pico-background-red-200">
          <strong>❌ {t('search.error')}:</strong> {error}
        </article>
      )}

      {loading ? (
        <figure>
          <table role="grid">
            <thead>
              <tr>
                <th scope="col">{t('requests.col.id')}</th>
                <th scope="col">{t('requests.col.target')}</th>
                <th scope="col">{t('requests.col.info')}</th>
                <th scope="col">{t('requests.col.status')}</th>
                {user?.role === 'admin' && <th scope="col">{t('requests.col.requestor')}</th>}
                {user?.role === 'admin' && <th scope="col">{t('requests.col.actions')}</th>}
              </tr>
            </thead>
            <tbody>
              {[1, 2, 3].map(i => <SkeletonRow key={i} />)}
            </tbody>
          </table>
        </figure>
      ) : (
        <figure>
          <table role="grid">
            <thead>
              <tr>
                <th scope="col">{t('requests.col.id')}</th>
                <th scope="col">{t('requests.col.target')}</th>
                <th scope="col">{t('requests.col.info')}</th>
                <th scope="col">{t('requests.col.status')}</th>
                {user?.role === 'admin' && <th scope="col">{t('requests.col.requestor')}</th>}
                {user?.role === 'admin' && <th scope="col">{t('requests.col.actions')}</th>}
              </tr>
            </thead>
            <tbody>
              {illRequests.length === 0 ? (
                <tr><td colSpan={user?.role === 'admin' ? 6 : 4} style={{ textAlign: 'center' }}>{t('requests.empty')}</td></tr>
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
                            <button className="outline" onClick={() => handleStatusUpdate(req.id, 'approved')} style={{ padding: '5px 10px', fontSize: '0.8em' }}>{t('requests.action.approve')}</button>
                            <button className="outline secondary" onClick={() => handleStatusUpdate(req.id, 'rejected')} style={{ padding: '5px 10px', fontSize: '0.8em' }}>{t('requests.action.reject')}</button>
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
