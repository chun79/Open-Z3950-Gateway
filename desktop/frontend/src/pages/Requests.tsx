import { useState, useEffect } from 'react'
import { Trash2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { ILLRequest } from '../types'
import { useI18n } from '../context/I18nContext'
import { ListILLRequests, DeleteILLRequest } from '../../wailsjs/go/main/App'
import { SkeletonRow } from '../components/Skeletons'

export default function Requests() {
  const [illRequests, setILLRequests] = useState<ILLRequest[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { t } = useI18n()

  const fetchILLRequests = async () => {
    setLoading(true)
    setError('')
    try {
      const data = await ListILLRequests()
      setILLRequests(data || [])
    } catch (err: any) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = (id: number) => {
    if (!confirm('Cancel this request?')) return
    toast.promise(
      DeleteILLRequest(id).then(() => fetchILLRequests()),
      {
        loading: 'Cancelling...',
        success: 'Request cancelled',
        error: 'Failed to cancel'
      }
    )
  }

  useEffect(() => {
    fetchILLRequests()
  }, [])

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
                <th scope="col">{t('requests.col.actions')}</th>
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
                <th scope="col">{t('requests.col.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {illRequests.length === 0 ? (
                <tr><td colSpan={5} style={{ textAlign: 'center' }}>{t('requests.empty')}</td></tr>
              ) : (
                illRequests.map((req, idx) => (
                  <tr key={idx}>
                    <td>{req.id}</td>
                    <td><small>{req.target_db}</small></td>
                    <td>
                      <strong>{req.title}</strong><br/>
                      <small>by {req.author} (ISBN: {req.isbn})</small>
                      {req.comments && (
                        <div style={{ marginTop: '5px', fontSize: '0.85em', color: '#666', fontStyle: 'italic' }}>
                          📝 {req.comments}
                        </div>
                      )}
                    </td>
                    <td>{getStatusBadge(req.status || 'pending')}</td>
                    <td>
                      {(!req.status || req.status === 'pending') && (
                        <button 
                          className="outline secondary"
                          onClick={() => req.id && handleDelete(req.id)}
                          style={{ border: 'none', padding: '4px', display: 'flex', alignItems: 'center' }}
                          title="Cancel Request"
                        >
                          <Trash2 size={16} />
                        </button>
                      )}
                    </td>
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
