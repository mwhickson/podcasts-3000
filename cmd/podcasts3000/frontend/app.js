import * as App from "./wailsjs/go/podcasts/App.js";

const state = { podcast: null, episodes: [] };
const $ = (selector) => document.querySelector(selector);
const escapeHTML = (value) => {
  const element = document.createElement("div");
  element.textContent = value || "";
  return element.innerHTML;
};

async function loadPodcasts() {
  const podcasts = await App.ListPodcasts();
  $("#podcasts").innerHTML = podcasts.map((podcast) =>
    `<div class="podcast ${state.podcast?.id === podcast.id ? "active" : ""}" data-id="${podcast.id}"><span class="podcast-name">${escapeHTML(podcast.title)}</span><button class="compact-button" data-purge-podcast="${podcast.id}" title="Purge this podcast's downloads">Purge</button></div>`
  ).join("");
  document.querySelectorAll(".podcast").forEach((element) => {
    element.onclick = () => selectPodcast(Number(element.dataset.id));
  });
  document.querySelectorAll("[data-purge-podcast]").forEach((button) => {
    button.onclick = async (event) => {
      event.stopPropagation();
      if (confirm("Delete downloads for this podcast?")) await App.PurgeCache("podcast", Number(button.dataset.purgePodcast));
    };
  });
  if (!state.podcast && podcasts.length) await selectPodcast(podcasts[0].id);
}

async function selectPodcast(id) {
  const podcasts = await App.ListPodcasts();
  state.podcast = podcasts.find((podcast) => podcast.id === id);
  state.episodes = await App.ListEpisodes(id);
  $("#active-podcast").innerHTML = `<div class="active-podcast-header">${state.podcast.artworkUrl ? `<img src="${escapeHTML(state.podcast.artworkUrl)}" alt="" class="artwork">` : `<div class="artwork artwork-placeholder">P</div>`}<div><p class="eyebrow">Now browsing</p><h2>${escapeHTML(state.podcast.title)}</h2><p class="muted">${escapeHTML(state.podcast.description)}</p></div></div>`;
  $("#episodes").innerHTML = state.episodes.length ? state.episodes.map((episode) =>
    `<article class="episode"><div class="episode-main"><div class="episode-heading"><h3>${episode.isFavorite ? "★ " : ""}${escapeHTML(episode.title)}</h3><div class="episode-actions"><button class="pill-button" data-play="${episode.id}">Play</button><button class="pill-button" data-favorite="${episode.id}">${episode.isFavorite ? "Unstar" : "Star"}</button><button class="pill-button" data-purge-episode="${episode.id}">Purge</button></div></div><p>${escapeHTML(episode.description)}</p><textarea data-note="${episode.id}" placeholder="Add a note">${escapeHTML(episode.notes)}</textarea><button class="text-button" data-save-note="${episode.id}">Save note</button></div></article>`
  ).join("") : `<p class="muted">No episodes found in this feed.</p>`;
  document.querySelectorAll("[data-play]").forEach((button) => {
    button.onclick = () => play(Number(button.dataset.play));
  });
  document.querySelectorAll("[data-favorite]").forEach((button) => {
    button.onclick = async () => { await App.ToggleFavorite(Number(button.dataset.favorite)); await selectPodcast(id); };
  });
  document.querySelectorAll("[data-save-note]").forEach((button) => {
    button.onclick = async () => {
      const episodeID = Number(button.dataset.saveNote);
      await App.SaveNote(episodeID, document.querySelector(`[data-note="${episodeID}"]`).value);
      await selectPodcast(id);
    };
  });
  document.querySelectorAll("[data-purge-episode]").forEach((button) => {
    button.onclick = async () => { if (confirm("Delete this downloaded episode?")) { await App.PurgeCache("episode", Number(button.dataset.purgeEpisode)); await selectPodcast(id); } };
  });
  await loadPodcasts();
}

async function play(id) {
  const episode = state.episodes.find((item) => item.id === id);
  const source = await App.GetAudioSource(id);
  $("#audio").src = source;
  $("#now-playing").textContent = episode.title;
  $("#audio").play();
}

$("#subscribe").onsubmit = async (event) => {
  event.preventDefault();
  try {
    await App.Subscribe($("#feed-url").value);
    $("#feed-url").value = "";
    await loadPodcasts();
  } catch (error) {
    alert(error);
  }
};

$("#refresh-all").onclick = async () => {
  await App.RefreshAllFeeds();
  await loadPodcasts();
  if (state.podcast) await selectPodcast(state.podcast.id);
};
$("#purge-all").onclick = async () => { if (confirm("Delete all downloaded media?")) await App.PurgeCache("all", 0); };
$("#itunes-search").onsubmit = async (event) => {
  event.preventDefault();
  try {
    const results = await App.SearchITunes($("#search-query").value);
    $("#search-view").hidden = false;
    $("#podcasts").hidden = true;
    $("#search-results").innerHTML = results.length ? results.map((result) =>
      `<div class="search-result"><strong>${escapeHTML(result.title)}</strong><button data-search-feed="${escapeHTML(result.feedUrl)}">Subscribe</button></div>`
    ).join("") : `<p class="muted">No podcasts found.</p>`;
    document.querySelectorAll("[data-search-feed]").forEach((button) => {
      button.onclick = async () => { await App.Subscribe(button.dataset.searchFeed); $("#search-results").replaceChildren(); await loadPodcasts(); };
    });
  } catch (error) { alert(error); }
};
$("#close-search").onclick = () => { $("#search-view").hidden = true; $("#podcasts").hidden = false; $("#search-results").replaceChildren(); $("#search-query").value = ""; };
$("#import-opml").onclick = async () => { try { await App.ImportOPMLDialog(); await loadPodcasts(); } catch (error) { alert(error); } };
$("#export-opml").onclick = async () => { try { await App.ExportOPMLDialog(); } catch (error) { alert(error); } };

$("#audio").addEventListener("timeupdate", () => {
  const episode = state.episodes.find((item) => item.title === $("#now-playing").textContent);
  if (episode && Math.floor($("#audio").currentTime) % 5 === 0) {
    App.SaveProgress(episode.id, Math.floor($("#audio").currentTime), false);
  }
});
$("#audio").addEventListener("ended", () => {
  const episode = state.episodes.find((item) => item.title === $("#now-playing").textContent);
  if (episode) App.SaveProgress(episode.id, Math.floor($("#audio").duration), true);
});

loadPodcasts().catch((error) => { $("#episodes").innerHTML = `<p class="muted">${escapeHTML(String(error))}</p>`; });
