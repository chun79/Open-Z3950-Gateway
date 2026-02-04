import { Book } from '../types';

export const generateBibTeX = (book: Book): string => {
  const key = book.isbn ? `isbn${book.isbn.replace(/[^0-9]/g, '')}` : `book${Date.now()}`;
  const author = book.author || 'Unknown';
  const title = book.title || 'Unknown Title';
  const publisher = book.publisher || '';
  const year = book.pub_year || '';
  const isbn = book.isbn || '';

  return `@book{${key},
  author = {${author}},
  title = {${title}},
  publisher = {${publisher}},
  year = {${year}},
  isbn = {${isbn}}
}`;
};

export const generateRIS = (book: Book): string => {
  const lines = ['TY  - BOOK'];
  if (book.title) lines.push(`TI  - ${book.title}`);
  if (book.author) lines.push(`AU  - ${book.author}`);
  if (book.publisher) lines.push(`PB  - ${book.publisher}`);
  if (book.pub_year) lines.push(`PY  - ${book.pub_year}`);
  if (book.isbn) lines.push(`SN  - ${book.isbn}`);
  if (book.issn) lines.push(`SN  - ${book.issn}`);
  lines.push('ER  - ');
  return lines.join('\n');
};
