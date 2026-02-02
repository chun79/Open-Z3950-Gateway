import React from 'react'

export const SkeletonCard = () => (
  <article>
    <div style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
      <div style={{ flexShrink: 0, width: '100px', height: '150px', backgroundColor: '#e0e0e0', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} />
      <div style={{ flexGrow: 1 }}>
        <div style={{ width: '60%', height: '24px', backgroundColor: '#e0e0e0', marginBottom: '10px', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} />
        <div style={{ width: '40%', height: '16px', backgroundColor: '#e0e0e0', marginBottom: '8px', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} />
        <div style={{ width: '30%', height: '16px', backgroundColor: '#e0e0e0', marginBottom: '8px', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} />
        <div style={{ width: '80%', height: '16px', backgroundColor: '#e0e0e0', marginTop: '10px', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} />
      </div>
    </div>
    <footer style={{ marginTop: '15px' }}>
      <div style={{ width: '100px', height: '30px', backgroundColor: '#e0e0e0', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} />
    </footer>
    <style>{`
      @keyframes pulse {
        0% { opacity: 1; }
        50% { opacity: 0.5; }
        100% { opacity: 1; }
      }
    `}</style>
  </article>
)

export const SkeletonRow = () => (
  <tr>
    <td><div style={{ width: '30px', height: '20px', backgroundColor: '#e0e0e0', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} /></td>
    <td><div style={{ width: '80px', height: '20px', backgroundColor: '#e0e0e0', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} /></td>
    <td><div style={{ width: '150px', height: '20px', backgroundColor: '#e0e0e0', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} /></td>
    <td><div style={{ width: '100px', height: '20px', backgroundColor: '#e0e0e0', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} /></td>
    <td><div style={{ width: '60px', height: '20px', backgroundColor: '#e0e0e0', borderRadius: '4px', animation: 'pulse 1.5s infinite' }} /></td>
  </tr>
)
