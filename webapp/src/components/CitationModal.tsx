import React from 'react'
import { useI18n } from '../context/I18nContext'

interface CitationModalProps {
  content: string
  format: string
  onClose: () => void
}

export function CitationModal({ content, format, onClose }: CitationModalProps) {
  const { t } = useI18n()

  return (
    <dialog open>
      <article>
        <header>
          <button aria-label="Close" rel="prev" onClick={onClose}></button>
          <strong>{t('search.citation.title').replace('{format}', format)}</strong>
        </header>
        <pre style={{ backgroundColor: '#f4f4f4', padding: '10px', borderRadius: '5px' }}>
          {content}
        </pre>
        <footer>
          <button 
            onClick={() => {
              navigator.clipboard.writeText(content)
              onClose()
            }}
          >
            {t('search.action.copy')}
          </button>
        </footer>
      </article>
    </dialog>
  )
}
