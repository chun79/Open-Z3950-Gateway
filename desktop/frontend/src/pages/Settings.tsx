import React, { useState, useEffect } from 'react'
import { Server, Database, Trash2, Play, Plus, Settings as SettingsIcon } from 'lucide-react'
import toast from 'react-hot-toast'
import { useI18n } from '../context/I18nContext'
import { ListTargets, AddTarget, DeleteTarget, TestTarget } from '../../wailsjs/go/main/App'

interface Target {
  name: string
  host: string
  port: number
  db: string
  encoding: string
}

export default function Settings() {
  const [targets, setTargets] = useState<Target[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const { t } = useI18n()

  // Form State
  const [newName, setNewName] = useState('')
  const [newHost, setNewHost] = useState('')
  const [newPort, setNewPort] = useState(210)
  const [newDB, setNewDB] = useState('')
  const [newEncoding, setNewEncoding] = useState('MARC21')

  const fetchTargets = async () => {
    setLoading(true)
    try {
      const data = await ListTargets()
      setTargets(data || [])
    } catch (err: any) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = (name: string) => {
    if (!confirm('Are you sure you want to delete this target?')) return
    toast.promise(
      DeleteTarget(name).then(() => fetchTargets()),
      {
        loading: 'Deleting...',
        success: 'Target deleted',
        error: 'Failed to delete'
      }
    )
  }

  const handleTest = async (host: string, port: number) => {
    const toastId = toast.loading('Testing connection...')
    try {
      const result = await TestTarget(host, port)
      if (result.success) {
        toast.success(result.message, { id: toastId })
      } else {
        toast.error(result.message, { id: toastId })
      }
    } catch (err: any) {
      toast.error("Request failed: " + String(err), { id: toastId })
    }
  }

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    toast.promise(
      (async () => {
        await AddTarget({
          name: newName,
          host: newHost,
          port: Number(newPort),
          db: newDB,
          encoding: newEncoding
        })
        setNewName('')
        setNewHost('')
        setNewPort(210)
        setNewDB('')
        fetchTargets()
      })(),
      {
        loading: 'Adding target...',
        success: 'Target added successfully',
        error: (err) => `Failed: ${err}`
      }
    )
  }

  useEffect(() => {
    fetchTargets()
  }, [])

  return (
    <article>
      <header style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <SettingsIcon size={24} />
        <strong>{t('settings.title')}</strong>
      </header>
      
      {error && <article className="pico-background-red-200">❌ {error}</article>}

      <figure>
        <table role="grid">
          <thead>
            <tr>
              <th>{t('settings.col.name')}</th>
              <th><div style={{display:'flex', alignItems:'center', gap:'5px'}}><Server size={14}/> {t('settings.col.host')}</div></th>
              <th><div style={{display:'flex', alignItems:'center', gap:'5px'}}><Database size={14}/> {t('settings.col.db')}</div></th>
              <th>{t('settings.col.encoding')}</th>
              <th>{t('settings.col.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {targets.map((target, idx) => (
              <tr key={idx}>
                <td><strong>{target.name}</strong></td>
                <td><small>{target.host}:{target.port}</small></td>
                <td>{target.db}</td>
                <td><mark>{target.encoding}</mark></td>
                <td>
                  <div role="group" style={{ marginBottom: 0 }}>
                    <button 
                      className="outline secondary" 
                      onClick={() => handleTest(target.host, target.port)} 
                      style={{ padding: '4px 8px', fontSize: '0.8em', display: 'flex', alignItems: 'center', gap: '4px' }}
                      title="Test Connection"
                    >
                      <Play size={14} /> {t('settings.btn.test')}
                    </button>
                    <button 
                      className="outline contrast" 
                      onClick={() => handleDelete(target.name)} 
                      style={{ padding: '4px 8px', fontSize: '0.8em', display: 'flex', alignItems: 'center', gap: '4px' }}
                      title="Delete"
                    >
                      <Trash2 size={14} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </figure>

      <hr />
      
      <h5><div style={{display:'flex', alignItems:'center', gap:'5px'}}><Plus size={18}/> {t('settings.add.title')}</div></h5>
      <form onSubmit={handleAdd}>
        <div className="grid">
          <label>{t('settings.add.name')} <input value={newName} onChange={e => setNewName(e.target.value)} placeholder="e.g. British Library" required /></label>
          <label>{t('settings.add.host')} <input value={newHost} onChange={e => setNewHost(e.target.value)} placeholder="z3950.bl.uk" required /></label>
          <label>{t('settings.add.port')} <input type="number" value={newPort} onChange={e => setNewPort(Number(e.target.value))} required /></label>
        </div>
        <div className="grid">
          <label>{t('settings.add.db')} <input value={newDB} onChange={e => setNewDB(e.target.value)} placeholder="main" required /></label>
          <label>{t('settings.add.encoding')} 
            <select value={newEncoding} onChange={e => setNewEncoding(e.target.value)}>
              <option value="MARC21">MARC21 (USMARC)</option>
              <option value="UNIMARC">UNIMARC</option>
              <option value="CNMARC">CNMARC</option>
            </select>
          </label>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-end' }}>
            <button type="submit">{t('settings.add.submit')}</button>
          </div>
        </div>
      </form>
    </article>
  )
}

