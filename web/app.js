async function fetchJSON(url){
  const r = await fetch(url);
  if(!r.ok) throw new Error(r.statusText);
  return r.json();
}
async function loadStats(){
  try{
    const s = await fetchJSON('/api/stats');
    document.getElementById('stats').textContent = `total: ${s.total} · drops: ${s.drops}`;
  }catch(e){ console.error(e); }
}
async function loadEvents(){
  const q = document.getElementById('search').value;
  const drops = document.getElementById('showDrops').checked ? '&drops=1' : '';
  const data = await fetchJSON('/api/events?limit=200&offset=0&q=' + encodeURIComponent(q) + drops);
  const tbl = document.getElementById('events');
  tbl.innerHTML = '<tr><th>ID</th><th>Time (UTC)</th><th>Host</th><th>Level</th><th>Message</th><th>IP</th></tr>';
  for(const ev of data.items){
    const tr = document.createElement('tr');
    tr.innerHTML = `<td>${ev.id}</td><td>${new Date(ev.received_at).toISOString()}</td><td>${ev.host||''}</td>
      <td><span class="badge info">${ev.level||''}</span></td>
      <td>${(ev.message||'').replace(/</g,'&lt;')}</td><td>${ev.source_ip||''}</td>`;
    tbl.appendChild(tr);
  }
}
loadStats(); loadEvents();
setInterval(()=>{loadStats();}, 5000);
