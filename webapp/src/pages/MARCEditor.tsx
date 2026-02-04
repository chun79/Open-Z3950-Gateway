import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useI18n } from '../context/I18nContext'
import { Book, MARCField } from '../types'
import { SkeletonCard } from '../components/Skeletons'
import toast from 'react-hot-toast'

export default function MARCEditor() {
  const { db, id } = useParams<{ db: string, id: string }>()
  const { token } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const [fields, setFields] = useState<MARCField[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    fetch(`/api/books/${db}/${encodeURIComponent(id || '')}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
      .then(res => res.json())
      .then(data => {
        if (data.status === 'success' && data.data.fields) {
          setFields(data.data.fields)
        }
        setLoading(false)
      })
      .catch(err => {
        console.error(err)
        setLoading(false)
      })
  }, [db, id, token])

  const handleFieldChange = (index: number, value: string) => {
    const newFields = [...fields]
    newFields[index].Value = value
    setFields(newFields)
  }

  const addField = () => {
    setFields([...fields, { Tag: '999', Value: '' }])
  }

  const removeField = (index: number) => {
    setFields(fields.filter((_, i) => i !== index))
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const res = await fetch(`/api/books/${db}/${encodeURIComponent(id || '')}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ fields })
      })
      if (res.ok) {
        toast.success("Metadata updated successfully")
        navigate(`/book/${db}/${id}`)
      } else {
        toast.error("Failed to save changes")
      }
    } catch (err) {
      toast.error("Network error")
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <SkeletonCard />

  return (
    <article>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <strong>MARC Metadata Editor</strong>
        <div role="group">
          <button className="secondary outline" onClick={() => navigate(-1)} style={{ marginBottom: 0 }}>Cancel</button>
          <button onClick={handleSave} disabled={saving} style={{ marginBottom: 0 }}>{saving ? 'Saving...' : 'Save Record'}</button>
        </div>
      </header>

      <table className="striped">
        <thead>
          <tr>
            <th style={{ width: '80px' }}>Tag</th>
            <th>Value</th>
            <th style={{ width: '50px' }}></th>
          </tr>
        </thead>
        <tbody>
          {fields.map((field, idx) => (
            <tr key={idx}>
              <td>
                <input 
                  type="text" 
                  value={field.Tag} 
                  onChange={(e) => {
                    const nf = [...fields]; nf[idx].Tag = e.target.value; setFields(nf);
                  }}
                  style={{ marginBottom: 0, padding: '2px 5px', textAlign: 'center', fontWeight: 'bold' }}
                />
              </td>
              <td>
                <input 
                  type="text" 
                  value={field.Value} 
                  onChange={(e) => handleFieldChange(idx, e.target.value)}
                  style={{ marginBottom: 0, padding: '2px 10px' }}
                />
              </td>
              <td>
                <button 
                  className="outline secondary" 
                  onClick={() => removeField(idx)}
                  style={{ border: 'none', padding: '5px', marginBottom: 0 }}
                >
                  ✕
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <button className="secondary outline" onClick={addField} style={{ width: '100%', marginTop: '10px' }}>
        + Add New Field
      </button>
    </article>
  )
}
