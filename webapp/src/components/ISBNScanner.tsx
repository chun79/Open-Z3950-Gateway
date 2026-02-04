import React, { useEffect, useState } from 'react'
import { Html5QrcodeScanner } from 'html5-qrcode'
import { useNavigate } from 'react-router-dom'

interface Props {
  onScanSuccess?: (isbn: string) => void
  onClose: () => void
}

export default function ISBNScanner({ onScanSuccess, onClose }: Props) {
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const scanner = new Html5QrcodeScanner(
      "reader",
      { 
        fps: 10, 
        qrbox: { width: 250, height: 150 }, // Aspect ratio for ISBN barcodes
        aspectRatio: 1.777778
      },
      /* verbose= */ false
    )

    scanner.render(
      (decodedText) => {
        // Clean ISBN (remove non-alphanumeric)
        const isbn = decodedText.replace(/[^0-9X]/gi, '')
        scanner.clear()
        if (onScanSuccess) {
          onScanSuccess(isbn)
        } else {
          navigate(`/?query=${isbn}&attr1=7`) // Jump to ISBN search
        }
        onClose()
      },
      (err) => {
        // Just log scanner errors, don't show to user constantly
      }
    )

    return () => {
      scanner.clear().catch(console.error)
    }
  }, [navigate, onClose, onScanSuccess])

  return (
    <dialog open style={{ width: '90%', maxWidth: '500px' }}>
      <article>
        <header>
          <a href="#close" aria-label="Close" className="close" onClick={onClose}></a>
          <strong>📷 Scan Book ISBN</strong>
        </header>
        <div id="reader" style={{ width: '100%' }}></div>
        {error && <p style={{ color: 'red' }}>{error}</p>}
        <footer>
          <small>Point your camera at the barcode on the back of the book.</small>
        </footer>
      </article>
    </dialog>
  )
}
