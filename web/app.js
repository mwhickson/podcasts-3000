const state = { podcast: null, episodes: [] };
const $ = (s) => document.querySelector(s);
async function api(path, options) { const response = await fetch(path, options); if (!response.ok) throw new Error(await response.text()); return response.json(); }
function escapeHTML(value) { const div = document.createElement('div'); div.textContent = value || ''; return div.innerHTML; }
async function loadPodcasts() {
  const podcasts = await api('/api/podcasts');
  $('#podcasts').innerHTML = podcasts.map(p => `<div class="podcast ${state.podcast?.id === p.id ? 'active' : ''}" data-id="${p.id}">${escapeHTML(p.title)}</div>`).join('');
  document.querySelectorAll('.podcast').forEach(el => el.onclick = () => selectPodcast(Number(el.dataset.id)));
}
async function selectPodcast(id) { state.podcast = (await api('/api/podcasts')).find(p => p.id === id); await loadEpisodes(id); await loadPodcasts(); }
async function loadEpisodes(id) {
  const episodes = await api(`/api/podcasts/${id}/episodes`); state.episodes = episodes;
  $('#episodes').innerHTML = episodes.map(ep => `<article class="episode"><div><h3>${ep.isFavorite ? '★ ' : ''}${escapeHTML(ep.title)}</h3><p>${escapeHTML(ep.description)}</p></div><button data-episode="${ep.id}">Play</button></article>`).join('');
  document.querySelectorAll('[data-episode]').forEach(button => button.onclick = () => play(Number(button.dataset.episode)));
}
async function play(id) { const ep = state.episodes.find(item => item.id === id); const source = await api(`/api/episodes/${id}/source`); $('#audio').src = source.source; $('#now-playing').textContent = ep.title; $('#audio').play(); }
$('#subscribe').onsubmit = async (event) => { event.preventDefault(); try { await api('/api/subscribe', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({url:$('#feed-url').value})}); $('#feed-url').value=''; await loadPodcasts(); } catch (error) { alert(error.message); } };
$('#refresh-all').onclick = async () => { await api('/api/refresh', {method:'POST'}); await loadPodcasts(); if (state.podcast) await loadEpisodes(state.podcast.id); };
loadPodcasts().catch(error => $('#episodes').innerHTML = `<p class="muted">${escapeHTML(error.message)}</p>`);
