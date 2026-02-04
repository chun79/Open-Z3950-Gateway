import React, { useState, useRef, useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import toast from 'react-hot-toast'

export default function CirculationDesk() {
  const { token } = useAuth()
  const [mode, setMode] = useState<'checkout' | 'checkin'>('checkout')
  const [patronId, setPatronId] = useState('')
  const [barcode, setBarcode] = useState('')
  const [log, setLog] = useState<string[]>([])
  
  const barcodeRef = useRef<HTMLInputElement>(null)

  const addToLog = (msg: string) => {
    setLog(prev => [`[${new Date().toLocaleTimeString()}] ${msg}`, ...prev].slice(0, 20))
  }

  const handleTransaction = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!barcode) return

    try {
      let endpoint = '/api/circulation/checkout'
      let body: any = { barcode }
      
      if (mode === 'checkout') {
        if (!patronId) {
          toast.error("Patron ID required for checkout")
          return
        }
        body.patron_id = patronId
      } else {
        endpoint = '/api/circulation/checkin'
      }

      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify(body)
      })
      
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'Transaction failed')

      if (mode === 'checkout') {
        toast.success(`Checked out! Due: ${new Date(data.due_date).toLocaleDateString()}`)
        addToLog(`OUT: Item ${barcode} to ${patronId} (Due: ${data.due_date})`)
      } else {
        const fineMsg = data.fine > 0 ? ` (Fine: $${data.fine})` : ''
        toast.success(`Checked in!${fineMsg}`)
        addToLog(`IN: Item ${barcode}${fineMsg}`)
      }
      
      setBarcode('')
      // Keep focus on barcode for rapid scanning
      barcodeRef.current?.focus()
    } catch (err: any) {
      toast.error(err.message)
      addToLog(`ERROR: ${err.message}`)
    }
  }

  return (
    <div className="container">
      <header style={{ marginBottom: '20px', borderBottom: '1px solid #ddd', paddingBottom: '10px' }}>
        <h2 style={{ margin: 0 }}>🏦 Circulation Desk</h2>
        <div role="group" style={{ marginTop: '10px' }}>
          <button 
            className={mode === 'checkout' ? '' : 'outline'} 
            onClick={() => setMode('checkout')}
          >
            Check Out (Loan)
          </button>
          <button 
            className={mode === 'checkin' ? '' : 'outline'} 
            onClick={() => setMode('checkin')}
          >
            Check In (Return)
          </button>
        </div>
      </header>

      <div className="grid">
        <div>
          <form onSubmit={handleTransaction} style={{ padding: '20px', background: '#f8f9fa', borderRadius: '8px' }}>
            {mode === 'checkout' && (
              <label>
                Patron Card / ID
                <input 
                  type="text" 
                  value={patronId} 
                  onChange={e => setPatronId(e.target.value)} 
                  placeholder="Scan Patron Card..."
                  autoFocus
                />
              </label>
            )}
            
            <label>
              Item Barcode
              <input 
                ref={barcodeRef}
                type="text" 
                value={barcode} 
                onChange={e => setBarcode(e.target.value)} 
                placeholder="Scan Book Barcode..."
                style={{ fontSize: '1.5em', padding: '15px' }}
              />
            </label>

            <button type="submit" style={{ width: '100%', fontSize: '1.2em' }}>
              {mode === 'checkout' ? 'Check Out Item' : 'Check In Item'}
            </button>
          </form>
        </div>

        <div style={{ padding: '20px', background: '#2d3748', color: '#fff', borderRadius: '8px', height: '400px', overflowY: 'auto' }}>
          <h5 style={{ borderBottom: '1px solid #555', paddingBottom: '5px' }}>Transaction Log</h5>
          <ul style={{ listStyle: 'none', padding: 0, fontFamily: 'monospace' }}>
            {log.map((l, i) => (
              <li key={i} style={{ marginBottom: '5px', color: l.includes('ERROR') ? '#fc8181' : l.includes('IN:') ? '#68d391' : '#63b3ed' }}>
                {l}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}
