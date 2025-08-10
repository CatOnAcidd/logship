async function fetchJSON(url){
  const r = await fetch(url);
  if(!r.ok) throw new Error('http '+r.status);
  return r.json();
}

function fmt(ts){
  const d = new Date(ts);
  return d.toLocaleString();
}

async function loadStats(){
  const s = await fetchJSON('/api/stats');
  document.getElementById('total').textContent = s.total;
  document.getElementById('hour').textContent = s.last_hour;
  document.getElementById('dropped').textContent = s.dropped;
}

async function loadEvents(){
  const q = document.getElementById('search').value.trim();
  const showDropped = document.getElementById('showDropped').checked;
  const url = new URL('/api/events', location.origin);
  if(q) url.searchParams.set('q', q);
  if(showDropped) url.searchParams.set('dropped', '1');
  const evs = await fetchJSON(url);
  const tbody = document.querySelector('#events tbody');
  tbody.innerHTML = '';
  for(const e of evs){
    const tr = document.createElement('tr');
    tr.innerHTML = `
      <td>${fmt(e.ts)}</td>
      <td>${e.host||''}</td>
      <td>${e.level||''}</td>
      <td>${(e.message||'').replace(/</g,'&lt;')}</td>
      <td>${e.source_ip||''}</td>
      <td>${e.dropped ? '✔' : ''}</td>
    `;
    tbody.appendChild(tr);
  }
}

document.getElementById('refresh').addEventListener('click', () => {
  loadStats().catch(console.error);
  loadEvents().catch(console.error);
});

document.getElementById('search').addEventListener('keydown', (e)=>{
  if(e.key === 'Enter') document.getElementById('refresh').click();
});

setInterval(()=>document.getElementById('refresh').click(), 5000);
document.getElementById('refresh').click();
