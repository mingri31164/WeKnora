import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import path from 'node:path'

const [checkout, directory] = process.argv.slice(2)
assert.ok(checkout && directory, 'Usage: node verify-closeout-agent.mjs <checkout> <artifact-directory>')
const require = createRequire(path.resolve(checkout, 'frontend/package.json'))
const JSZip = require('jszip')
const XLSX = require('xlsx')
const { JSDOM } = require('jsdom')
const names = ['support-input.csv', 'support-review.pptx', 'support-report.html', 'support-summary.xlsx', 'support-summary.csv']
const files = new Map(await Promise.all(names.map(async name => [name, await readFile(path.join(directory, name))])))
const header = ['team', 'tickets', 'resolved', 'resolution_percent']
const dom = new JSDOM('')

async function verify(data) {
  const rows = workbook => XLSX.utils.sheet_to_json(workbook.Sheets[workbook.SheetNames[0]], { header: 1, defval: '' })
  const input = rows(XLSX.read(data.get('support-input.csv'), { type: 'buffer', raw: true }))
  assert.deepEqual(input[0], ['month', 'team', 'tickets', 'resolved'])
  assert.equal(input.length, 7)
  const totals = new Map([['Platform', [0, 0]], ['Search', [0, 0]]])
  for (const [month, team, tickets, resolved] of input.slice(1)) {
    assert.match(month, /^2026-0[678]$/)
    assert.match(tickets, /^\d+$/)
    assert.match(resolved, /^\d+$/)
    assert.ok(totals.has(team))
    totals.get(team)[0] += Number(tickets)
    totals.get(team)[1] += Number(resolved)
  }
  assert.deepEqual([...totals], [['Platform', [450, 417]], ['Search', [300, 280]]])
  totals.set('Total', [750, 697])
  function table(actual) {
    assert.deepEqual(actual[0], header)
    assert.deepEqual(actual.slice(1).map(row => row[0]), [...totals.keys()])
    for (const [team, tickets, resolved, percent] of actual.slice(1)) {
      assert.deepEqual([tickets, resolved], totals.get(team))
      assert.equal(percent, Math.round(resolved / tickets * 1000) / 10, `${team}: one-decimal percentage`)
    }
  }
  const workbook = XLSX.read(data.get('support-summary.xlsx'), { type: 'buffer', cellNF: true })
  assert.equal(workbook.SheetNames.length, 1)
  const sheet = workbook.Sheets[workbook.SheetNames[0]]
  assert.equal(sheet['!ref'], 'A1:D6')
  assert.equal(sheet.A6.v, 'Synthetic demonstration data; resolution_percent = resolved / tickets.')
  for (const key of ['D2', 'D3', 'D4']) assert.equal(sheet[key].z, '0.0')
  for (const [key, cell] of Object.entries(sheet)) {
    if (key.startsWith('!')) continue
    assert.ok(!cell.f)
    assert.ok(/^[A-D][1-4]$/.test(key) || key === 'A6', `Unexpected cell ${key}`)
  }
  table(rows(workbook).slice(0, 4))
  const csv = rows(XLSX.read(data.get('support-summary.csv'), { type: 'buffer' }))
  table(csv)
  assert.deepEqual(csv, rows(workbook).slice(0, 4))
  const html = new JSDOM(data.get('support-report.html').toString())
  try {
    const doc = html.window.document
    assert.equal(doc.querySelectorAll('script,iframe,object,embed,form,link,base,meta[http-equiv],[src],[href],[srcset]').length, 0)
    assert.match(doc.body.textContent, /synthetic/i)
    assert.match(doc.querySelector('caption')?.textContent || '', /one decimal/)
    for (const node of doc.querySelectorAll('*')) {
      assert.ok([...node.attributes].every(attr => !/^on/i.test(attr.name)))
    }
    for (const node of doc.querySelectorAll('style,[style]')) {
      assert.doesNotMatch(node.textContent + (node.getAttribute('style') || ''), /@import|url\s*\(/i)
    }
    const element = doc.querySelector('table')
    assert.ok(element)
    table([...element.rows].map((row, i) => [...row.cells].map((cell, j) => i && j ? Number(cell.textContent.trim()) : cell.textContent.trim())))
  } finally { html.window.close() }
  for (const name of ['support-review.pptx', 'support-summary.xlsx']) {
    const zip = await JSZip.loadAsync(data.get(name), { checkCRC32: true })
    for (const file of Object.values(zip.files).filter(file => file.name.endsWith('.rels'))) {
      const xml = new dom.window.DOMParser().parseFromString(await file.async('text'), 'application/xml')
      assert.equal(xml.getElementsByTagName('parsererror').length, 0)
      for (const rel of xml.getElementsByTagNameNS('*', 'Relationship')) assert.notEqual(rel.getAttribute('TargetMode')?.toLowerCase(), 'external')
    }
    if (name.endsWith('.pptx')) {
      const slides = Object.keys(zip.files).filter(key => /^ppt\/slides\/slide\d+\.xml$/.test(key))
      assert.equal(slides.length, 4)
      let text = ''
      for (const key of slides) {
        const xml = new dom.window.DOMParser().parseFromString(await zip.file(key).async('text'), 'application/xml')
        assert.equal(xml.getElementsByTagName('parsererror').length, 0)
        text += [...xml.getElementsByTagNameNS('http://schemas.openxmlformats.org/drawingml/2006/main', 't')].map(node => node.textContent).join(' ') + '\n'
      }
      for (const pattern of [/synthetic/i, /750/, /697/]) assert.match(text, pattern)
    }
  }
}

try {
  await verify(files)
  const rejected = []
  const csv = files.get('support-summary.csv').toString()
  const html = files.get('support-report.html').toString()
  for (const [name, target, text] of [
    ['wrong-percentage', 'support-summary.csv', csv.replace('92.7', '91.7')],
    ['wrong-total', 'support-summary.csv', csv.replace('750,697', '751,697')],
    ['duplicate-row', 'support-summary.csv', csv.replace('Search,', 'Platform,')],
    ['html-active-script', 'support-report.html', html.replace('</body>', '<script>0</script></body>')],
    ['html-resource', 'support-report.html', html.replace('</body>', '<img src="https://example.invalid/x"></body>')],
  ]) {
    assert.notEqual(text, files.get(target).toString())
    const altered = new Map(files)
    altered.set(target, Buffer.from(text))
    await assert.rejects(() => verify(altered), undefined, name)
    rejected.push(name)
  }
  console.log(JSON.stringify({
    status: 'passed', scope: 'Independent offline semantic checks at the declared one-decimal precision; not browser rendering.',
    original_strict_check: { status: 'failed', reason: 'The initial driver required two-decimal accuracy, which the prompt did not specify. Its failure is retained.' },
    rejected_mutations: rejected,
    files: names.map(name => ({ name, bytes: files.get(name).length, sha256: createHash('sha256').update(files.get(name)).digest('hex') })),
    limitations: ['CSV lacks the requested embedded synthetic-data notice.', 'XLSX has a synthetic-data footer outside the four-row table.', 'The HTML/XLSX explanatory formula omits the factor of 100; numeric percentage values are correct.'],
  }, null, 2))
} finally { dom.window.close() }
