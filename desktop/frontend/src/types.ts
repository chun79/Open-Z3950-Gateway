export interface Book {
  title: string
  author: string
  isbn: string
  issn?: string
  subject?: string
  pub_year?: string
  publisher?: string
  source?: string
  record_id?: string
  summary?: string
  toc?: string
  edition?: string
  physical?: string
  series?: string
  notes?: string
  holdings?: Holding[]
  // Additional fields for BookDetail
  fields?: any[]
}

export interface Holding {
  call_number: string
  status: string
  location: string
}

export interface ILLRequest {
  id?: number
  target_db: string
  title: string
  author: string
  isbn: string
  status?: string
  requestor?: string
  comments?: string
  created_at?: string
}
