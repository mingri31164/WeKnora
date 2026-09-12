import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const args = process.argv.slice(2).filter(arg => arg !== '--self-test')
assert.ok(args.length >= 1 && args.length <= 2,
  'Usage: node verify-agent-artifacts.mjs <WeKnora-checkout> [artifacts-directory] [--self-test]')
const require = createRequire(path.resolve(args[0], 'frontend/package.json'))
const JSZip = require('jszip')
const XLSX = require('xlsx')
const { JSDOM } = require('jsdom')
const directory = path.resolve(args[1] || fileURLToPath(new URL('./artifacts/', import.meta.url)))
const names = [
  'support-input.csv', 'support-review.pptx', 'support-report.html',
  'support-summary.xlsx', 'support-summary.csv',
]
const files = new Map()
for (const name of names) {
  const bytes = await readFile(path.join(directory, name))
  assert.ok(bytes.length > 0 && bytes.length <= 8 * 1024 * 1024, `${name}: size`)
  files.set(name, bytes)
}
const header = ['team', 'tickets', 'resolved', 'resolution_percent']
const tableRows = workbook => XLSX.utils.sheet_to_json(
  workbook.Sheets[workbook.SheetNames[0]], { header: 1, defval: '' },
)
const spreadsheet = bytes => XLSX.read(bytes, { type: 'buffer' })

async function verify(data) {
  const input = tableRows(XLSX.read(data.get('support-input.csv'), { type: 'buffer', raw: true }))
  assert.deepEqual(input[0], ['month', 'team', 'tickets', 'resolved'])
  assert.equal(input.length, 7, 'Six input rows')
  const totals = new Map([['Platform', [0, 0]], ['Search', [0, 0]]])
  for (const [month, team, ticketsText, resolvedText] of input.slice(1)) {
    assert.match(month, /^2026-0[678]$/)
    assert.match(ticketsText, /^\d+$/)
    assert.match(resolvedText, /^\d+$/)
    const tickets = Number(ticketsText)
    const resolved = Number(resolvedText)
    assert.ok(totals.has(team) && Number.isInteger(tickets) && Number.isInteger(resolved))
    assert.ok(tickets > 0 && resolved >= 0 && resolved <= tickets)
    const total = totals.get(team)
    total[0] += tickets
    total[1] += resolved
  }
  assert.deepEqual([...totals], [['Platform', [450, 417]], ['Search', [300, 280]]])
  totals.set('Total', [750, 697])
  function verifyRows(rows) {
    assert.deepEqual(rows[0], header)
    assert.deepEqual(rows.slice(1).map(row => row[0]), [...totals.keys()], 'Row identities')
    for (const [team, tickets, resolved, percent] of rows.slice(1)) {
      assert.deepEqual([tickets, resolved], totals.get(team), `${team}: totals`)
      assert.ok(Math.abs(Number(percent) - resolved / tickets * 100) < 0.011,
        `${team}: percentage`)
    }
  }
  const book = spreadsheet(data.get('support-summary.xlsx'))
  assert.equal(book.SheetNames.length, 1)
  const rows = tableRows(book)
  verifyRows(rows)
  for (const [key, cell] of Object.entries(book.Sheets[book.SheetNames[0]])) {
    if (!key.startsWith('!')) assert.ok(!cell.f, 'Literal values only')
  }
  assert.deepEqual(tableRows(spreadsheet(data.get('support-summary.csv'))), rows,
    'CSV and XLSX must match')

  const dom = new JSDOM('')
  const html = new JSDOM(data.get('support-report.html').toString('utf8'))
  try {
    const parseXML = text => {
      const xml = new dom.window.DOMParser().parseFromString(text, 'application/xml')
      assert.equal(xml.getElementsByTagName('parsererror').length, 0, 'Valid XML')
      return xml
    }
    for (const name of ['support-review.pptx', 'support-summary.xlsx']) {
      const zip = await JSZip.loadAsync(data.get(name), { checkCRC32: true })
      for (const file of Object.values(zip.files).filter(file => file.name.endsWith('.rels'))) {
        const xml = parseXML(await file.async('text'))
        for (const node of xml.getElementsByTagNameNS('*', 'Relationship')) {
          assert.notEqual(node.getAttribute('TargetMode')?.toLowerCase(), 'external',
            `${name}: external relationship`)
        }
      }
      if (name.endsWith('.pptx')) {
        const slides = Object.keys(zip.files).filter(key => /^ppt\/slides\/slide\d+\.xml$/.test(key))
        assert.equal(slides.length, 4, 'Four PPTX slides')
        let text = ''
        for (const key of slides) {
          const xml = parseXML(await zip.file(key).async('text'))
          text += [...xml.getElementsByTagNameNS(
            'http://schemas.openxmlformats.org/drawingml/2006/main', 't',
          )].map(node => node.textContent).join(' ') + '\n'
        }
        for (const pattern of [/synthetic/i, /750/, /697/]) assert.match(text, pattern)
      }
    }
    const document = html.window.document
    assert.equal(document.querySelectorAll(
      'script,iframe,object,embed,form,link,base,meta[http-equiv],[src],[href],[srcset]',
    ).length, 0, 'Passive HTML')
    for (const node of document.querySelectorAll('*')) {
      assert.ok([...node.attributes].every(attr => !/^on/i.test(attr.name)),
        'No HTML event handlers')
    }
    for (const node of document.querySelectorAll('style,[style]')) {
      assert.doesNotMatch(node.textContent + (node.getAttribute('style') || ''),
        /@import|url\s*\(/i, 'No CSS resources')
    }
    assert.match(document.body.textContent, /synthetic/i)
    const table = document.querySelector('table')
    assert.ok(table, 'HTML table')
    verifyRows([...table.rows].map((row, index) => [...row.cells].map(
      (cell, column) => index && column
        ? Number(cell.textContent.trim().replace(/%$/, ''))
        : cell.textContent.trim(),
    )))
  } finally {
    html.window.close()
    dom.window.close()
  }
}

await verify(files)
const rejectedMutations = []
if (process.argv.includes('--self-test')) {
  const csv = files.get('support-summary.csv').toString()
  const html = files.get('support-report.html').toString()
  const mutations = [
    ['csv-mismatch', 'support-summary.csv', csv.replace('750,697', '751,697')],
    ['duplicate-row', 'support-summary.csv', csv.replace('Search,', 'Platform,')],
    ['html-wrong-total', 'support-report.html', html.replace('<td>750</td>', '<td>751</td>')],
    ['html-active-script', 'support-report.html', html.replace('</body>', '<script>0</script></body>')],
    ['html-resource', 'support-report.html', html.replace('</body>', '<img src="https://example.invalid/x"></body>')],
  ]
  for (const [label, name, text] of mutations) {
    assert.notEqual(text, files.get(name).toString(), `Mutation applied: ${label}`)
    const altered = new Map(files)
    altered.set(name, Buffer.from(text))
    await assert.rejects(() => verify(altered), undefined, label)
    rejectedMutations.push(label)
  }
}
console.log(JSON.stringify({
  status: 'passed',
  scope: 'Offline structure and data checks, not browser rendering or Agent replay.',
  files: names.map(name => ({
    name, sha256: createHash('sha256').update(files.get(name)).digest('hex'),
  })),
  rejected_mutations: rejectedMutations,
  limitations: [
    'CSV/XLSX contain the required table but no embedded synthetic-data notice.',
    'This checks the supplied demonstration artifacts, not arbitrary untrusted documents.',
  ],
}, null, 2))
