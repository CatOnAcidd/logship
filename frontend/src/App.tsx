import React, { useEffect, useState } from 'react'
import { api } from './api'
import { Card } from './components'

type Log = { id:number; ts:string; source_ip:string; action:string; rule_id?:number; raw:string; size_bytes:number }
type Rule = { id:number; name:string; enabled:boolean; source_cidr?:string; hostname?:string; app_name?:string; facility?:string; severity?:string; message_regex?:string }
type Dashboard = { last24h_received:number; last24h_forwarded:number; last24h_dropped:number; last24h_unmatched:number; throughput_series:number[][] }

function Table({rows, onCreateFrom}:{rows:Log[]; onCreateFrom?:(log:Log)=>void}){
  return (
    <table>
      <thead>
        <tr><th>Time</th><th>Source</th><th>Rule</th><th>Action</th><th>Message</th><th></th></tr>
      </thead>
      <tbody>
        {rows.map(r => (
          <tr key={r.id}>
            <td className="muted">{new Date(r.ts).toLocaleString()}</td>
            <td>{r.source_ip}</td>
            <td>{r.rule_id ? <span className="pill">#{r.rule_id}</span> : '-'}</td>
            <td><span className="pill">{r.action}</span></td>
            <td style={{whiteSpace:'nowrap', textOverflow:'ellipsis', overflow:'hidden', maxWidth: 420}}>{r.raw}</td>
            <td>{onCreateFrom && <button className="btn" onClick={() => onCreateFrom(r)}>Create rule</button>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function Rules(){
  const [rules, setRules] = useState<Rule[]>([])
  const [form, setForm] = useState<Partial<Rule>>({ enabled: true })
  const load = async () => setRules((await api.get('/api/rules')).data)
  useEffect(() => { load() }, [])
  const save = async () => {
    await api.post('/api/rules', form)
    setForm({ enabled: true })
    await load()
  }
  const toggle = async (r:Rule) => {
    await api.put('/api/rules/'+r.id, {...r, enabled: !r.enabled})
    await load()
  }
  const del = async (r:Rule) => {
    await api.delete('/api/rules/'+r.id)
    await load()
  }
  return (
    <Card title="Rules" extra={null}>
      <div className="grid grid-4">
        <div>
          <label>Name</label>
          <input value={form.name||''} onChange={e=>setForm({...form, name:e.target.value})}/>
        </div>
        <div>
          <label>Source CIDR</label>
          <input value={form.source_cidr||''} onChange={e=>setForm({...form, source_cidr:e.target.value})}/>
        </div>
        <div>
          <label>Hostname</label>
          <input value={form.hostname||''} onChange={e=>setForm({...form, hostname:e.target.value})}/>
        </div>
        <div>
          <label>App-Name</label>
          <input value={form.app_name||''} onChange={e=>setForm({...form, app_name:e.target.value})}/>
        </div>
        <div>
          <label>Facility</label>
          <input value={form.facility||''} onChange={e=>setForm({...form, facility:e.target.value})}/>
        </div>
        <div>
          <label>Severity</label>
          <input value={form.severity||''} onChange={e=>setForm({...form, severity:e.target.value})}/>
        </div>
        <div className="grid" style={{gridTemplateColumns:'1fr'}}>
          <label>Message Regex</label>
          <input value={form.message_regex||''} onChange={e=>setForm({...form, message_regex:e.target.value})}/>
        </div>
        <div className="row" style={{alignItems:'center'}}>
          <label>Enabled</label>
          <input type="checkbox" checked={!!form.enabled} onChange={e=>setForm({...form, enabled:e.target.checked})}/>
          <button className="btn" onClick={save} style={{marginLeft: 'auto'}}>Add Rule</button>
        </div>
      </div>
      <div style={{height:8}}/>
      <table>
        <thead><tr><th>ID</th><th>Name</th><th>Enabled</th><th>Match</th><th></th></tr></thead>
        <tbody>
          {rules.map(r => (
            <tr key={r.id}>
              <td>#{r.id}</td>
              <td>{r.name}</td>
              <td><button className="btn" onClick={()=>toggle(r)}>{r.enabled ? 'Disable' : 'Enable'}</button></td>
              <td className="muted">{[r.source_cidr, r.hostname, r.app_name, r.facility, r.severity, r.message_regex].filter(Boolean).join(' • ') || '(any)'}</td>
              <td><button className="btn" onClick={()=>del(r)}>Delete</button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}

function Settings(){
  const [settings, setSettings] = useState<{key:string,value:string}[]>([])
  const load = async () => setSettings((await api.get('/api/settings')).data)
  useEffect(()=>{ load() },[])
  const save = async (key:string, value:string) => {
    await api.post('/api/settings', {key, value})
    await load()
  }
  const get = (k:string) => settings.find(s => s.key===k)?.value || ''
  const onTest = async () => {
    const res = await api.post('/api/test-destination', null, { params: { host:get('DEST_HOST'), port:get('DEST_PORT'), protocol:get('DEST_PROTOCOL') }})
    alert('Destination test: ' + (res.data.ok ? 'OK' : 'FAILED'))
  }
  return (
    <Card title="Settings">
      <div className="grid grid-2">
        <div className="card">
          <h4 className="title">Default Action</h4>
          <div className="row">
            <select value={get('DEFAULT_ACTION')} onChange={e=>save('DEFAULT_ACTION', e.target.value)}>
              <option value="BLOCK">Block</option>
              <option value="FORWARD">Forward</option>
            </select>
            <span className="muted">Default is Block.</span>
          </div>
        </div>
        <div className="card">
          <h4 className="title">Destination</h4>
          <div className="grid">
            <div><label>Host</label><input value={get('DEST_HOST')} onChange={e=>save('DEST_HOST', e.target.value)} /></div>
            <div><label>Port</label><input value={get('DEST_PORT')} onChange={e=>save('DEST_PORT', e.target.value)} /></div>
            <div><label>Protocol</label>
              <select value={get('DEST_PROTOCOL')} onChange={e=>save('DEST_PROTOCOL', e.target.value)}>
                <option>udp</option><option>tcp</option>
              </select>
            </div>
            <div><button className="btn" onClick={onTest}>Test Destination</button></div>
          </div>
        </div>
        <div className="card">
          <h4 className="title">Storage</h4>
          <div className="grid">
            <div><label>Max Temp Storage (MB)</label><input value={get('MAX_TEMP_STORAGE_MB')} onChange={e=>save('MAX_TEMP_STORAGE_MB', e.target.value)} /></div>
            <div><label>Max Dropped Storage (MB)</label><input value={get('MAX_DROPPED_STORAGE_MB')} onChange={e=>save('MAX_DROPPED_STORAGE_MB', e.target.value)} /></div>
            <div><label>Max Threshold Reached Storage (MB)</label><input value={get('MAX_THRESHOLD_REACHED_STORAGE_MB')} onChange={e=>save('MAX_THRESHOLD_REACHED_STORAGE_MB', e.target.value)} /></div>
          </div>
        </div>
        <div className="card">
          <h4 className="title">Forwarding Threshold</h4>
          <div className="grid">
            <div><label>Enabled</label>
              <select value={get('THRESHOLD_ENABLED')} onChange={e=>save('THRESHOLD_ENABLED', e.target.value)}>
                <option value="false">false</option><option value="true">true</option>
              </select>
            </div>
            <div><label>Max Bytes</label><input value={get('THRESHOLD_BYTES')} onChange={e=>save('THRESHOLD_BYTES', e.target.value)} /></div>
            <div><label>Period</label>
              <select value={get('THRESHOLD_PERIOD')} onChange={e=>save('THRESHOLD_PERIOD', e.target.value)}>
                <option value="1d">1 Day</option>
                <option value="7d">7 Days</option>
                <option value="30d">30 Days</option>
                <option value="PT12H">12 Hours</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </Card>
  )
}

function Dashboard(){
  const [dash, setDash] = useState<Dashboard|null>(null)
  const [recent, setRecent] = useState<any>({forwarded:[], dropped:[], unmatched:[], received:[]})
  const load = async () => {
    setDash((await api.get('/api/dashboard')).data)
    setRecent((await api.get('/api/logs/recent')).data)
  }
  useEffect(()=>{ load(); const t = setInterval(load, 5000); return ()=>clearInterval(t) }, [])
  return (
    <div>
      <div className="grid grid-4">
        <Card title="Received (24h)">{dash?.last24h_received ?? '-'}</Card>
        <Card title="Forwarded (24h)">{dash?.last24h_forwarded ?? '-'}</Card>
        <Card title="Dropped (24h)">{dash?.last24h_dropped ?? '-'}</Card>
        <Card title="Unmatched (24h)">{dash?.last24h_unmatched ?? '-'}</Card>
      </div>
      <div className="grid grid-2">
        <Card title="Recent Forwarded"><Table rows={recent.forwarded}/></Card>
        <Card title="Recent Dropped"><Table rows={recent.dropped} onCreateFrom={(log)=>alert('Open rule wizard prefilled from sample: '+log.id)}/></Card>
      </div>
      <div className="grid grid-2">
        <Card title="Recent Unmatched"><Table rows={recent.unmatched} onCreateFrom={(log)=>alert('Open rule wizard prefilled from sample: '+log.id)}/></Card>
        <Card title="Recent Received"><Table rows={recent.received}/></Card>
      </div>
    </div>
  )
}

export default function App(){
  const [tab, setTab] = useState<'dashboard'|'rules'|'settings'>('dashboard')
  return (
    <div className="container">
      <div className="row" style={{justifyContent:'space-between'}}>
        <h2 className="title">Logship Syslog Filter</h2>
        <div className="row">
          <button className="btn" onClick={()=>setTab('dashboard')}>Dashboard</button>
          <button className="btn" onClick={()=>setTab('rules')}>Rules</button>
          <button className="btn" onClick={()=>setTab('settings')}>Settings</button>
        </div>
      </div>
      {tab==='dashboard' && <Dashboard/>}
      {tab==='rules' && <Rules/>}
      {tab==='settings' && <Settings/>}
    </div>
  )
}
